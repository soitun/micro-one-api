package conf

import "testing"

func TestSubscriptionConfigDefaultDisabled(t *testing.T) {
	var cfg *Subscription
	if cfg.GetSubscriptionEnabled() {
		t.Fatal("subscription should be disabled by default")
	}
}

func TestSubscriptionConfigEnabled(t *testing.T) {
	cfg := &Subscription{Enabled: true}
	if !cfg.GetSubscriptionEnabled() {
		t.Fatal("subscription should be enabled")
	}
}

func TestSubscriptionConfigUserRPMLimitDefault(t *testing.T) {
	var cfg *Subscription
	if got := cfg.GetUserRPMLimit(); got != 0 {
		t.Fatalf("user rpm limit default = %d, want 0", got)
	}
	cfg = &Subscription{UserRpmLimit: 12}
	if got := cfg.GetUserRPMLimit(); got != 12 {
		t.Fatalf("user rpm limit = %d, want 12", got)
	}
	cfg = &Subscription{UserRpmLimit: -1}
	if got := cfg.GetUserRPMLimit(); got != -1 {
		t.Fatalf("user rpm limit = %d, want -1", got)
	}
}

func TestTokenRefreshConfigDefaultsToHybridFlag(t *testing.T) {
	var cfg *HybridAdaptor
	if cfg.GetTokenRefreshEnabled() {
		t.Fatal("token refresh should be disabled when hybrid adaptor is disabled")
	}
	cfg = &HybridAdaptor{Enabled: true}
	if !cfg.GetTokenRefreshEnabled() {
		t.Fatal("token refresh should follow hybrid adaptor enabled by default")
	}
}

func TestTokenRefreshNestedConfig(t *testing.T) {
	cfg := &HybridAdaptor{
		RefreshInterval:  "10m",
		RefreshLookahead: "24h",
		TokenRefresh: &TokenRefresh{
			Enabled:                  true,
			CheckIntervalMinutes:     5,
			RefreshBeforeExpiryHours: 12,
		},
	}
	if got := cfg.GetTokenRefreshEnabled(); !got {
		t.Fatal("explicit token refresh flag should enable refresh")
	}
}

func TestOpenaiWSDrainTimeoutDefault(t *testing.T) {
	var cfg *OpenaiWS
	if got := cfg.GetOpenAIWSDrainTimeout(); got != "30s" {
		t.Fatalf("drain timeout default = %q, want 30s", got)
	}
	cfg = &OpenaiWS{DrainTimeout: "45s"}
	if got := cfg.GetOpenAIWSDrainTimeout(); got != "45s" {
		t.Fatalf("drain timeout = %q, want 45s", got)
	}
}
