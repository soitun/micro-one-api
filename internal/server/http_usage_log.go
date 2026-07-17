package server

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	logv1 "micro-one-api/api/log/v1"
	applogger "micro-one-api/platform/logging"
	"micro-one-api/platform/metrics"
)

type usageLogInput struct {
	UserID                int64
	TokenID               int64
	TokenName             string
	RequestID             string
	Endpoint              string
	ModelName             string
	Quota                 int64
	PromptTokens          int64
	CompletionTokens      int64
	CacheReadTokens       int64
	ChannelID             int64
	SubscriptionAccountID int64
	Group                 string
	SessionHash           string
	SessionWindowLimitUSD float64
	ElapsedTime           int64
	IsStream              bool
}

func (s *HTTPServer) ingestUsageLog(ctx context.Context, in usageLogInput) {
	if s.logClient == nil {
		metrics.UsageLogIngestTotal.WithLabelValues("skipped").Inc()
		return
	}
	message := applogger.Sanitize(fmt.Sprintf("model=%s quota=%d prompt_tokens=%d completion_tokens=%d cache_read_tokens=%d channel=%d", in.ModelName, in.Quota, in.PromptTokens, in.CompletionTokens, in.CacheReadTokens, in.ChannelID))
	_, err := s.logClient.IngestLog(ctx, &logv1.IngestLogRequest{
		Level:                 "consume",
		Message:               message,
		Source:                "relay-gateway",
		RequestId:             in.RequestID,
		UserId:                in.UserID,
		TokenName:             usageTokenName(in),
		ModelName:             in.ModelName,
		Quota:                 in.Quota,
		PromptTokens:          in.PromptTokens,
		CompletionTokens:      in.CompletionTokens,
		CacheReadTokens:       in.CacheReadTokens,
		ChannelId:             in.ChannelID,
		SubscriptionAccountId: in.SubscriptionAccountID,
		ElapsedTime:           in.ElapsedTime,
		IsStream:              in.IsStream,
	})
	if err != nil && applogger.Log != nil {
		metrics.UsageLogIngestTotal.WithLabelValues("error").Inc()
		applogger.Log.Warn("failed to ingest usage log", zap.Error(err))
		return
	}
	metrics.UsageLogIngestTotal.WithLabelValues("success").Inc()
}

func logUpstreamUsage(in usageLogInput) {
	cacheRatio := float64(0)
	if in.PromptTokens > 0 {
		cacheRatio = float64(in.CacheReadTokens) / float64(in.PromptTokens)
	}
	nonCachedInputTokens := in.PromptTokens
	if in.CacheReadTokens > 0 {
		nonCachedInputTokens = in.PromptTokens - in.CacheReadTokens
		if nonCachedInputTokens < 0 {
			nonCachedInputTokens = 0
		}
	}
	applogger.Log.Info("upstream usage reported",
		zap.String("request_id", in.RequestID),
		zap.String("endpoint", in.Endpoint),
		zap.String("model", in.ModelName),
		zap.Int64("user_id", in.UserID),
		zap.Int64("channel_id", in.ChannelID),
		zap.Bool("is_stream", in.IsStream),
		zap.Int64("total_tokens", in.Quota),
		zap.Int64("upstream_input_tokens", in.PromptTokens),
		zap.Int64("input_tokens", nonCachedInputTokens),
		zap.Int64("output_tokens", in.CompletionTokens),
		zap.Int64("cache_read_tokens", in.CacheReadTokens),
		zap.Float64("cache_read_input_ratio", cacheRatio),
	)
}
