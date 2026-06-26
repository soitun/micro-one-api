package server

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/bytedance/sonic"

	relayadaptor "micro-one-api/internal/relay/adaptor"
	relaybiz "micro-one-api/internal/relay/biz"
	relaycredential "micro-one-api/internal/relay/credential"
	relayprovider "micro-one-api/internal/relay/provider"
)

// handleChatCompletionsViaAdaptor is the feature-flag-gated request path for
// subscription-account channels (Codex / Claude OAuth). It resolves the real
// subscription-account metadata from the selected channel, builds a
// RelayContext, and drives the full ConvertRequest → BuildUpstreamRequest →
// upstream call → ConvertResponse / ConvertStreamResponse pipeline.
//
// It is intentionally a thin, self-contained path: it does not participate in
// the RetryExecutor (subscription accounts are selected explicitly and retried
// via a different mechanism in later phases), and it performs only
// best-effort quota accounting. The goal of the MVP is to validate that the
// Responses-hub + mimicry + credential layers compose end-to-end.
func (s *HTTPServer) handleChatCompletionsViaAdaptor(
	w http.ResponseWriter,
	r *http.Request,
	plan *relaybiz.RelayPlan,
	clientModel string,
	rawBody []byte,
) {
	s.handleSubscriptionAccountViaAdaptor(w, r, plan, clientModel, rawBody, relayadaptor.FormatOpenAIChatCompletions)
}

func (s *HTTPServer) handleAnthropicMessagesViaAdaptor(
	w http.ResponseWriter,
	r *http.Request,
	plan *relaybiz.RelayPlan,
	clientModel string,
	rawBody []byte,
) {
	s.handleSubscriptionAccountViaAdaptor(w, r, plan, clientModel, rawBody, relayadaptor.FormatAnthropicMessages)
}

func (s *HTTPServer) handleResponsesCreateLikeViaAdaptor(
	w http.ResponseWriter,
	r *http.Request,
	plan *relaybiz.RelayPlan,
	clientModel string,
	rawBody []byte,
) {
	s.handleSubscriptionAccountViaAdaptor(w, r, plan, clientModel, rawBody, relayadaptor.FormatOpenAIResponses)
}

func (s *HTTPServer) handleSubscriptionAccountViaAdaptor(
	w http.ResponseWriter,
	r *http.Request,
	plan *relaybiz.RelayPlan,
	clientModel string,
	rawBody []byte,
	inbound relayadaptor.Format,
) {
	if plan == nil || plan.Channel == nil {
		s.writeError(w, http.StatusInternalServerError, "no channel selected")
		return
	}

	ad, ok := relayadaptor.GetAdaptor(plan.Channel.Type)
	if !ok {
		s.writeError(w, http.StatusBadGateway, "no adaptor registered for subscription channel type")
		return
	}

	meta := fallbackSubscriptionAccountMetadata(plan, plan.Channel)
	if s.accountResolver != nil {
		if resolved, err := s.accountResolver.Resolve(r.Context(), plan.Channel.ID); err == nil && resolved != nil {
			meta = resolved
		}
	}

	// Build the relay context with the real account identity. Account.ID keys
	// the credential/identity caches; AccountID is the upstream account id
	// (chatgpt-account-id / Claude metadata user_id).
	rc := &relayadaptor.RelayContext{
		InboundFormat: inbound,
		ClientModel:   clientModel,
		ResolvedModel: plan.ResolvedModel,
		Channel:       plan.Channel,
		Account: &relayadaptor.AccountRef{
			ID:          meta.ID,
			Platform:    string(meta.Platform),
			AccountType: "oauth",
			GroupID:     meta.GroupID,
			AccountID:   meta.AccountID,
			AccessToken: meta.AccessToken,
			Fingerprint: meta.Fingerprint,
		},
		UserID:        plan.Auth.UserID,
		InboundHeader: r.Header.Clone(),
		RawBody:       rawBody,
	}
	// Carry the configured upstream HTTP client so the OAuth path respects the
	// gateway's timeout/proxy/transport settings instead of silently falling
	// back to http.DefaultClient.
	if s.oauthHTTPClient != nil {
		rc.HTTPClient = s.oauthHTTPClient
	}
	ad.Init(rc)

	// Convert the inbound request body to the upstream format.
	upstreamFmt, upstreamBody, err := ad.ConvertRequest(rc, inbound, rawBody)
	if err != nil {
		s.writeError(w, http.StatusBadGateway, fmt.Sprintf("adaptor convert request: %v", err))
		return
	}

	// Build the upstream http.Request (includes identity mimicry + OAuth token).
	upstreamReq, err := ad.BuildUpstreamRequest(r.Context(), rc, upstreamFmt, upstreamBody)
	if err != nil {
		s.writeError(w, http.StatusBadGateway, fmt.Sprintf("adaptor build request: %v", err))
		return
	}

	// Determine whether the client requested streaming.
	isStream := false
	var probe map[string]any
	if err := sonic.Unmarshal(rawBody, &probe); err == nil {
		if v, ok := probe["stream"].(bool); ok {
			isStream = v
		}
	}

	// Use the relay context's client (configured timeout/transport) rather than
	// http.DefaultClient so the OAuth path inherits the gateway's upstream
	// settings. Fall back to DefaultClient only when none is configured.
	client := rc.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(upstreamReq)
	if err != nil {
		s.writeError(w, http.StatusBadGateway, fmt.Sprintf("upstream call: %v", err))
		return
	}
	defer resp.Body.Close()

	// Non-2xx: forward the upstream error body to the client (best-effort).
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		s.writeError(w, resp.StatusCode, fmt.Sprintf("upstream: %s", strings.TrimSpace(string(body))))
		return
	}

	if isStream {
		_, reader, err := ad.ConvertStreamResponse(rc, upstreamFmt, resp)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("adaptor convert stream: %v", err))
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)
		if flusher, ok := w.(http.Flusher); ok {
			_, _ = io.Copy(&flushWriter{w: w, flusher: flusher}, reader)
			return
		}
		_, _ = io.Copy(w, reader)
		return
	}

	// Non-streaming: convert and write.
	_, outBody, err := ad.ConvertResponse(rc, upstreamFmt, resp)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("adaptor convert response: %v", err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(outBody)
}

func fallbackSubscriptionAccountMetadata(plan *relaybiz.RelayPlan, ch *relaybiz.Channel) *relaycredential.SubscriptionAccountMetadata {
	meta := &relaycredential.SubscriptionAccountMetadata{
		ID:       ch.ID,
		GroupID:  ch.Group,
		Platform: relaycredential.PlatformClaude,
	}
	meta.AccountID = fmt.Sprintf("%d", ch.ID)
	switch ch.Type {
	case relayprovider.ChannelTypeCodexOAuth:
		meta.Platform = relaycredential.PlatformCodex
	case relayprovider.ChannelTypeClaudeOAuth:
		meta.Platform = relaycredential.PlatformClaude
	}
	meta.AccessToken = ch.Key
	return meta
}

// platformTagFromChannelType maps a subscription channel type to its platform
// tag. It uses the provider package's channel-type constants directly.
func platformTagFromChannelType(t int32) string {
	switch t {
	case relayprovider.ChannelTypeCodexOAuth:
		return "codex"
	case relayprovider.ChannelTypeClaudeOAuth:
		return "claude"
	default:
		return ""
	}
}
