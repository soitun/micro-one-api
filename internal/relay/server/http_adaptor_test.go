package server

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	relaybiz "micro-one-api/internal/relay/biz"
	relaycredential "micro-one-api/internal/relay/credential"
	relayprovider "micro-one-api/internal/relay/provider"
)

type testSubscriptionResolver struct {
	meta *relaycredential.SubscriptionAccountMetadata
}

func (r testSubscriptionResolver) Resolve(context.Context, int64) (*relaycredential.SubscriptionAccountMetadata, error) {
	if r.meta == nil {
		return nil, relaycredential.ErrAccountNotFound
	}
	cp := *r.meta
	return &cp, nil
}

func TestHandleChatCompletionsViaAdaptor_UsesFallbackMetadata(t *testing.T) {
	httpServer := NewHTTPServer(nil, nil, nil, nil, nil)
	httpServer.SetHybridAdaptorEnabled(true)
	var seenAuth, seenAccountID string
	httpServer.SetOAuthHTTPClient(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			seenAuth = req.Header.Get("Authorization")
			seenAccountID = req.Header.Get("chatgpt-account-id")
			return newJSONResponse(`{"id":"resp_1","object":"response","model":"gpt-5","status":"completed","output":[{"type":"message","id":"msg_1","role":"assistant","content":[{"type":"output_text","text":"ok"}]}]}`), nil
		}),
	})
	httpServer.SetSubscriptionAccountResolver(testSubscriptionResolver{meta: nil})

	plan := &relaybiz.RelayPlan{
		Auth: &relaybiz.AuthSnapshot{UserID: 42, Group: "default"},
		Channel: &relaybiz.Channel{
			ID:      11,
			Type:    relayprovider.ChannelTypeCodexOAuth,
			BaseURL: "https://example.invalid",
			Key:     "acct-123",
			Group:   "default",
		},
		ResolvedModel: "gpt-5",
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-5","messages":[{"role":"user","content":"hi"}],"stream":false}`))
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()

	httpServer.handleChatCompletionsViaAdaptor(rec, req, plan, "gpt-5", []byte(`{"model":"gpt-5","messages":[{"role":"user","content":"hi"}],"stream":false}`))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"ok"`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
	if seenAuth != "Bearer acct-123" {
		t.Fatalf("Authorization = %q", seenAuth)
	}
	if seenAccountID != "11" {
		t.Fatalf("chatgpt-account-id = %q", seenAccountID)
	}
}

func TestHandleChatCompletionsViaAdaptor_PlanAccountWinsOverResolver(t *testing.T) {
	httpServer := NewHTTPServer(nil, nil, nil, nil, nil)
	httpServer.SetHybridAdaptorEnabled(true)
	var seenAuth, seenAccountID string
	httpServer.SetOAuthHTTPClient(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			seenAuth = req.Header.Get("Authorization")
			seenAccountID = req.Header.Get("chatgpt-account-id")
			return newJSONResponse(`{"id":"resp_1","object":"response","model":"gpt-5","status":"completed","output":[{"type":"message","id":"msg_1","role":"assistant","content":[{"type":"output_text","text":"ok"}]}]}`), nil
		}),
	})
	httpServer.SetSubscriptionAccountResolver(testSubscriptionResolver{meta: &relaycredential.SubscriptionAccountMetadata{
		ID:          99,
		Platform:    relaycredential.PlatformCodex,
		AccountType: "oauth",
		AccessToken: "resolver-token",
		AccountID:   "resolver-account",
	}})

	plan := &relaybiz.RelayPlan{
		Auth: &relaybiz.AuthSnapshot{UserID: 42, Group: "default"},
		Channel: &relaybiz.Channel{
			ID:      12,
			Type:    relayprovider.ChannelTypeCodexOAuth,
			BaseURL: "https://example.invalid",
			Group:   "default",
		},
		Account: &relaybiz.SubscriptionAccount{
			ID:          12,
			Platform:    "codex",
			AccountType: "oauth",
			AccessToken: "plan-token",
			AccountID:   "plan-account",
			Group:       "default",
		},
		ResolvedModel: "gpt-5",
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-5","messages":[{"role":"user","content":"hi"}],"stream":false}`))
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()

	httpServer.handleChatCompletionsViaAdaptor(rec, req, plan, "gpt-5", []byte(`{"model":"gpt-5","messages":[{"role":"user","content":"hi"}],"stream":false}`))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	if seenAuth != "Bearer plan-token" {
		t.Fatalf("Authorization = %q", seenAuth)
	}
	if seenAccountID != "plan-account" {
		t.Fatalf("chatgpt-account-id = %q", seenAccountID)
	}
}

func TestHandleChatCompletionsViaAdaptor_FailoverOnRetryableUpstreamStatus(t *testing.T) {
	selector := &adaptorFailoverChannelClient{
		accounts: []*relaybiz.SubscriptionAccount{
			{
				ID:          13,
				Name:        "second",
				Platform:    "codex",
				AccountType: "oauth",
				Status:      1,
				BaseURL:     "https://example.invalid",
				Group:       "default",
				Models:      []string{"gpt-5"},
				Priority:    10,
				AccessToken: "second-token",
				AccountID:   "second-account",
			},
		},
	}
	relayUsecase := relaybiz.NewRelayUsecase(adaptorFailoverIdentity{}, selector, nil, nil)
	httpServer := NewHTTPServer(nil, nil, nil, nil, relayUsecase)
	httpServer.SetHybridAdaptorEnabled(true)
	httpServer.wsPoolCfg.failoverMaxSwitches = 1

	var authHeaders []string
	httpServer.SetOAuthHTTPClient(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			authHeaders = append(authHeaders, req.Header.Get("Authorization"))
			if len(authHeaders) == 1 {
				return newStatusResponse(http.StatusInternalServerError, `{"error":"temporary"}`), nil
			}
			return newJSONResponse(`{"id":"resp_2","object":"response","model":"gpt-5","status":"completed","output":[{"type":"message","id":"msg_1","role":"assistant","content":[{"type":"output_text","text":"ok"}]}]}`), nil
		}),
	})

	plan := &relaybiz.RelayPlan{
		Auth: &relaybiz.AuthSnapshot{UserID: 42, Group: "default"},
		Channel: &relaybiz.Channel{
			ID:      12,
			Type:    relayprovider.ChannelTypeCodexOAuth,
			BaseURL: "https://example.invalid",
			Group:   "default",
		},
		Account: &relaybiz.SubscriptionAccount{
			ID:          12,
			Platform:    "codex",
			AccountType: "oauth",
			Status:      1,
			BaseURL:     "https://example.invalid",
			Group:       "default",
			Models:      []string{"gpt-5"},
			Priority:    20,
			AccessToken: "first-token",
			AccountID:   "first-account",
		},
		ResolvedModel: "gpt-5",
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-5","messages":[{"role":"user","content":"hi"}],"stream":false}`))
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()

	httpServer.handleChatCompletionsViaAdaptor(rec, req, plan, "gpt-5", []byte(`{"model":"gpt-5","messages":[{"role":"user","content":"hi"}],"stream":false}`))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	if len(authHeaders) != 2 {
		t.Fatalf("upstream calls = %d, want 2", len(authHeaders))
	}
	if authHeaders[0] != "Bearer first-token" || authHeaders[1] != "Bearer second-token" {
		t.Fatalf("Authorization headers = %v", authHeaders)
	}
	if metrics := httpServer.runtimeBlocker.Metrics(); metrics.Blocks != 1 {
		t.Fatalf("runtime blocks = %d, want 1", metrics.Blocks)
	}
	if selector.calls != 1 {
		t.Fatalf("selector calls = %d, want 1", selector.calls)
	}
}

type adaptorFailoverIdentity struct{}

func (adaptorFailoverIdentity) GetAuthSnapshot(context.Context, string) (*relaybiz.AuthSnapshot, error) {
	return &relaybiz.AuthSnapshot{UserID: 42, Group: "default"}, nil
}

type adaptorFailoverChannelClient struct {
	accounts []*relaybiz.SubscriptionAccount
	calls    int
}

func (c *adaptorFailoverChannelClient) SelectChannel(context.Context, string, string, bool) (*relaybiz.Channel, error) {
	return nil, errors.New("no api-key channel")
}

func (c *adaptorFailoverChannelClient) RecordChannelHealth(context.Context, int64, bool, string, int64) error {
	return nil
}

func (c *adaptorFailoverChannelClient) SelectSubscriptionAccount(_ context.Context, _, _, _ string, _ bool) (*relaybiz.SubscriptionAccount, error) {
	if c.calls >= len(c.accounts) {
		return nil, errors.New("no subscription account")
	}
	account := c.accounts[c.calls]
	c.calls++
	return account, nil
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func newJSONResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func newStatusResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
