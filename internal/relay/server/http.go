package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"micro-one-api/api/identity/v1"
	channelv1 "micro-one-api/api/channel/v1"
	billingv1 "micro-one-api/api/billing/v1"
	commonv1 "micro-one-api/api/common/v1"
	"micro-one-api/internal/pkg/errors"
	applogger "micro-one-api/internal/pkg/logger"
	relaybiz "micro-one-api/internal/relay/biz"
	relayprovider "micro-one-api/internal/relay/provider"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

// HTTPServer handles HTTP requests for relay-gateway.
type HTTPServer struct {
	identityClient  identityv1.IdentityServiceClient
	channelClient  channelv1.ChannelServiceClient
	billingClient  billingv1.BillingServiceClient
	providerFactory *relayprovider.ProviderFactory
	retry           *retryConfig
	modelMapper     *relaybiz.ModelMapper
}

// NewHTTPServer creates a new HTTP server for Kratos.
func NewHTTPServer(
	identityClient identityv1.IdentityServiceClient,
	channelClient channelv1.ChannelServiceClient,
	billingClient billingv1.BillingServiceClient,
	providerFactory *relayprovider.ProviderFactory,
) *HTTPServer {
	return &HTTPServer{
		identityClient:  identityClient,
		channelClient:   channelClient,
		billingClient:   billingClient,
		providerFactory: providerFactory,
	}
}

// SetRetryConfig sets the retry configuration for upstream provider calls.
func (s *HTTPServer) SetRetryConfig(maxAttempts int, initialInterval, maxInterval string, multiplier float64, retryableStatus []int) {
	s.retry = parseRetryConfig(maxAttempts, initialInterval, maxInterval, multiplier, retryableStatus)
}

// SetModelMapper sets the model name mapper for resolving client model names to upstream names.
func (s *HTTPServer) SetModelMapper(mapper *relaybiz.ModelMapper) {
	s.modelMapper = mapper
}

// RegisterRoutes registers HTTP routes to a Kratos *khttp.Server.
func (s *HTTPServer) RegisterRoutes(srv *khttp.Server) {
	srv.HandleFunc("/v1/chat/completions", s.handleChatCompletions)
	srv.HandleFunc("/v1/models", s.handleModels)
	srv.HandleFunc("/health", s.handleHealth)
}

func (s *HTTPServer) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		s.writeError(w, http.StatusUnauthorized, "missing authorization header")
		return
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		s.writeError(w, http.StatusUnauthorized, "invalid authorization header format")
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		s.writeError(w, http.StatusUnauthorized, "missing token")
		return
	}

	var req relayprovider.ChatCompletionsRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	if req.Model == "" {
		s.writeError(w, http.StatusBadRequest, "model is required")
		return
	}

	authSnapshot, err := s.getAuthSnapshot(r.Context(), token)
	if err != nil {
		s.handleIdentityError(w, err)
		return
	}

	if !s.isModelAllowed(authSnapshot.AllowedModels, req.Model) {
		s.writeError(w, http.StatusForbidden, "model not allowed")
		return
	}

	// Resolve model name mapping (e.g. gpt-4o -> gpt-4o-2024-08-06)
	if s.modelMapper != nil {
		req.Model = s.modelMapper.Resolve(req.Model)
	}

	// First channel selection
	channel, err := s.selectChannel(r.Context(), authSnapshot.Group, req.Model)
	if err != nil {
		s.handleChannelError(w, err)
		return
	}

	// Retry loop for upstream provider calls
	maxAttempts := 1
	if s.retry != nil {
		maxAttempts = s.retry.maxAttempts
	}

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Wait before retry (skip on first attempt)
		if attempt > 0 && s.retry != nil {
			wait := backoffDuration(attempt-1, s.retry.initialInterval, s.retry.maxInterval, s.retry.multiplier)
			logRetry(attempt-1, maxAttempts, wait, lastErr)
			time.Sleep(wait)

			// Re-select channel with excludeFirstPriority for fallback
			newChannel, selErr := s.selectChannelExcludePriority(r.Context(), authSnapshot.Group, req.Model, true)
			if selErr != nil {
				// No more channels available, return the last upstream error
				break
			}
			channel = newChannel
		}

		// Reserve quota
		requestID := generateRequestID()
		estimatedTokens := s.estimateTokens(&req)
		reservation, reserveErr := s.reserveQuota(r.Context(), fmt.Sprintf("%d", authSnapshot.UserId), requestID, estimatedTokens, req.Model, fmt.Sprintf("%d", channel.Id))
		if reserveErr != nil {
			s.handleBillingError(w, reserveErr)
			return
		}

		provider, provErr := s.providerFactory.CreateProvider(channel.Type, channel.BaseUrl, channel.Key)
		if provErr != nil {
			_ = s.releaseQuota(r.Context(), reservation.ReservationId, "failed to create provider")
			s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create provider: %v", provErr))
			return
		}

		if req.Stream {
			s.handleStreamingResponse(w, r, provider, &req, reservation)
			return
		}

		// Non-streaming call
		resp, callErr := provider.ChatCompletions(r.Context(), &req)
		if callErr != nil {
			_ = s.releaseQuota(r.Context(), reservation.ReservationId, "upstream error")
			lastErr = callErr
			// Check if retryable
			if s.retry == nil || !isRetryableError(callErr, s.retry.retryableStatus) {
				s.writeError(w, mapUpstreamError(upstreamStatus(callErr)), fmt.Sprintf("upstream error: %v", callErr))
				return
			}
			continue
		}

		// Success — commit quota and return
		actualTokens := s.calculateActualTokens(resp)
		_ = s.commitQuota(r.Context(), reservation.ReservationId, actualTokens, true)
		s.writeJSON(w, http.StatusOK, resp)
		return
	}

	// All retries exhausted
	if lastErr != nil {
		s.writeError(w, mapUpstreamError(upstreamStatus(lastErr)), fmt.Sprintf("upstream error after %d attempts: %v", maxAttempts, lastErr))
	} else {
		s.writeError(w, http.StatusServiceUnavailable, "no available channels")
	}
}

func (s *HTTPServer) handleStreamingResponse(w http.ResponseWriter, r *http.Request, provider relayprovider.Provider, req *relayprovider.ChatCompletionsRequest, reservation *billingv1.ReserveQuotaResponse) {
	chunkChan, err := provider.ChatCompletionsStream(r.Context(), req)
	if err != nil {
		// 流式请求失败，释放预扣配额
		_ = s.releaseQuota(r.Context(), reservation.ReservationId, "upstream stream error")
		s.writeError(w, http.StatusBadGateway, fmt.Sprintf("upstream stream error: %v", err))
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		// 流式不支持，释放预扣配额
		_ = s.releaseQuota(r.Context(), reservation.ReservationId, "streaming not supported")
		s.writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Transfer-Encoding", "chunked")

	totalTokens := int64(0)
	streamError := false

	for chunk := range chunkChan {
		// StreamChunk 没有 Usage 字段，我们需要估算 tokens
		// 这里简单使用字符数除以 4 来估算
		for _, choice := range chunk.Choices {
			totalTokens += int64(len(choice.Delta.Content) / 4)
		}

		jsonData, err := sonic.Marshal(chunk)
		if err != nil {
			applogger.Log.Warn("failed to marshal chunk", zap.Error(err))
			continue
		}

		fmt.Fprintf(w, "data: %s\n\n", string(jsonData))
		flusher.Flush()
	}

	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()

	// 流式请求完成，提交配额
	if !streamError {
		_ = s.commitQuota(r.Context(), reservation.ReservationId, totalTokens, true)
	} else {
		_ = s.releaseQuota(r.Context(), reservation.ReservationId, "stream error")
	}
}

func (s *HTTPServer) handleModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		s.writeError(w, http.StatusUnauthorized, "missing authorization header")
		return
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		s.writeError(w, http.StatusUnauthorized, "invalid authorization header format")
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		s.writeError(w, http.StatusUnauthorized, "missing token")
		return
	}

	authSnapshot, err := s.getAuthSnapshot(r.Context(), token)
	if err != nil {
		s.handleIdentityError(w, err)
		return
	}

	modelsReply, err := s.listAvailableModels(r.Context(), authSnapshot.Group)
	if err != nil {
		s.handleChannelError(w, err)
		return
	}

	models := s.applyModelWhitelist(modelsReply.Models, authSnapshot.AllowedModels)

	response := struct {
		Object string `json:"object"`
		Data   []struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int64  `json:"created"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}{
		Object: "list",
	}

	for _, model := range models {
		response.Data = append(response.Data, struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int64  `json:"created"`
			OwnedBy string `json:"owned_by"`
		}{
			ID:      model,
			Object:  "model",
			Created: 0,
			OwnedBy: "organization",
		})
	}

	s.writeJSON(w, http.StatusOK, response)
}

func (s *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *HTTPServer) getAuthSnapshot(ctx context.Context, token string) (*identityv1.GetAuthSnapshotReply, error) {
	req := &identityv1.GetAuthSnapshotRequest{
		Token: token,
	}
	return s.identityClient.GetAuthSnapshot(ctx, req)
}

func (s *HTTPServer) selectChannel(ctx context.Context, group, model string) (*commonv1.ChannelInfo, error) {
	req := &channelv1.SelectChannelRequest{
		Group:              group,
		Model:              model,
		ExcludeFirstPriority: false,
	}
	resp, err := s.channelClient.SelectChannel(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.Channel, nil
}

func (s *HTTPServer) selectChannelExcludePriority(ctx context.Context, group, model string, excludeFirst bool) (*commonv1.ChannelInfo, error) {
	req := &channelv1.SelectChannelRequest{
		Group:              group,
		Model:              model,
		ExcludeFirstPriority: excludeFirst,
	}
	resp, err := s.channelClient.SelectChannel(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.Channel, nil
}

func (s *HTTPServer) listAvailableModels(ctx context.Context, group string) (*channelv1.ListAvailableModelsReply, error) {
	req := &channelv1.ListAvailableModelsRequest{
		Group: group,
	}
	return s.channelClient.ListAvailableModels(ctx, req)
}

func (s *HTTPServer) isModelAllowed(allowedModels []string, model string) bool {
	if len(allowedModels) == 0 {
		return true
	}
	for _, allowed := range allowedModels {
		if allowed == model {
			return true
		}
	}
	return false
}

func (s *HTTPServer) applyModelWhitelist(availableModels []string, allowedModels []string) []string {
	if len(allowedModels) == 0 {
		return availableModels
	}

	allowedSet := make(map[string]bool)
	for _, model := range allowedModels {
		allowedSet[model] = true
	}

	filtered := make([]string, 0, len(availableModels))
	for _, model := range availableModels {
		if allowedSet[model] {
			filtered = append(filtered, model)
		}
	}

	return filtered
}

func (s *HTTPServer) handleIdentityError(w http.ResponseWriter, err error) {
	// Check for structured errors first
	if errors.IsUnauthorized(err) {
		s.writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	if errors.IsForbidden(err) {
		s.writeError(w, http.StatusForbidden, err.Error())
		return
	}

	// Handle gRPC errors
	st, ok := status.FromError(err)
	if ok {
		switch st.Code() {
		case codes.NotFound:
			s.writeError(w, http.StatusUnauthorized, st.Message())
		case codes.PermissionDenied:
			s.writeError(w, http.StatusForbidden, st.Message())
		case codes.ResourceExhausted:
			s.writeError(w, http.StatusTooManyRequests, st.Message())
		default:
			s.writeError(w, http.StatusInternalServerError, st.Message())
		}
		return
	}

	s.writeError(w, http.StatusInternalServerError, err.Error())
}

func (s *HTTPServer) handleChannelError(w http.ResponseWriter, err error) {
	if errors.IsServiceUnavailable(err) {
		s.writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	s.writeError(w, http.StatusInternalServerError, err.Error())
}

func (s *HTTPServer) handleBillingError(w http.ResponseWriter, err error) {
	// 处理计费服务错误
	st, ok := status.FromError(err)
	if ok {
		switch st.Code() {
		case codes.ResourceExhausted:
			// 配额不足
			s.writeError(w, http.StatusPaymentRequired, "insufficient quota")
		case codes.Unavailable:
			// 计费服务不可用
			s.writeError(w, http.StatusServiceUnavailable, "billing service unavailable")
		default:
			s.writeError(w, http.StatusInternalServerError, st.Message())
		}
		return
	}
	s.writeError(w, http.StatusInternalServerError, err.Error())
}

func (s *HTTPServer) writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	encodeJSON(w, map[string]interface{}{
		"error": map[string]interface{}{
			"message": message,
		},
	})
}

func (s *HTTPServer) writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	encodeJSON(w, data)
}

// 配额管理方法

func (s *HTTPServer) reserveQuota(ctx context.Context, userID, requestID string, estimatedTokens int64, model, channelID string) (*billingv1.ReserveQuotaResponse, error) {
	req := &billingv1.ReserveQuotaRequest{
		UserId:         userID,
		RequestId:      requestID,
		EstimatedTokens: estimatedTokens,
		Model:          model,
		ChannelId:      channelID,
	}
	return s.billingClient.ReserveQuota(ctx, req)
}

func (s *HTTPServer) commitQuota(ctx context.Context, reservationID string, actualTokens int64, success bool) error {
	req := &billingv1.CommitQuotaRequest{
		ReservationId: reservationID,
		ActualTokens:  actualTokens,
		Success:       success,
	}
	_, err := s.billingClient.CommitQuota(ctx, req)
	return err
}

func (s *HTTPServer) releaseQuota(ctx context.Context, reservationID, reason string) error {
	req := &billingv1.ReleaseQuotaRequest{
		ReservationId: reservationID,
		Reason:        reason,
	}
	_, err := s.billingClient.ReleaseQuota(ctx, req)
	return err
}

func (s *HTTPServer) estimateTokens(req *relayprovider.ChatCompletionsRequest) int64 {
	// 简单的 token 估算逻辑
	// 实际应用中可以使用更精确的 tokenizer
	tokens := int64(0)

	// 估算输入 tokens
	for _, msg := range req.Messages {
		tokens += int64(len(msg.Content) / 4) // 假设平均每个 token 4 个字符
	}

	// 估算输出 tokens (基于 max_tokens 或默认值)
	if req.MaxTokens != nil && *req.MaxTokens > 0 {
		tokens += int64(*req.MaxTokens)
	} else {
		tokens += 1000 // 默认输出 tokens
	}

	return tokens
}

func (s *HTTPServer) calculateActualTokens(resp *relayprovider.ChatCompletionsResponse) int64 {
	// resp.Usage 不是指针，是值类型
	return int64(resp.Usage.TotalTokens)
}

func generateRequestID() string {
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}

