package main

import (
	"strings"
	"testing"

	identitycfg "micro-one-api/internal/identity/config"
)

func TestRegistrationPolicyFromConfigDefaultsEnabled(t *testing.T) {
	policy := registrationPolicyFromConfig(&identitycfg.Config{})

	if !policy.Enabled {
		t.Fatal("registration should default to enabled")
	}
}

func TestRegistrationPolicyFromConfigSupportsRestrictionsAndExplicitDisable(t *testing.T) {
	policy := registrationPolicyFromConfig(&identitycfg.Config{
		Registration: identitycfg.RegistrationConfig{
			Disabled:                      true,
			EmailDomainRestrictionEnabled: true,
			EmailDomainWhitelist:          []string{"example.com"},
			TurnstileCheckEnabled:         true,
			TurnstileSecret:               "secret",
		},
	})

	if policy.Enabled {
		t.Fatal("registration should be disabled")
	}
	if !policy.EmailDomainRestrictionEnabled || policy.EmailDomainWhitelist[0] != "example.com" {
		t.Fatalf("email domain policy mismatch: %+v", policy)
	}
	if !policy.TurnstileCheckEnabled || policy.TurnstileSecret != "secret" {
		t.Fatalf("turnstile policy mismatch: %+v", policy)
	}
}

func TestSetupOAuthRegistersOIDCWhenConfigured(t *testing.T) {
	registry := setupOAuth(&identitycfg.Config{
		OAuth: identitycfg.OAuthConfig{
			BaseURL: "https://one-api.example.com",
			OIDC: identitycfg.OIDCProviderConfig{
				Enabled:      true,
				ClientID:     "client-id",
				ClientSecret: "client-secret",
				AuthorizeURL: "https://idp.example.com/oauth2/authorize",
				TokenURL:     "https://idp.example.com/oauth2/token",
				UserInfoURL:  "https://idp.example.com/oauth2/userinfo",
				Scopes:       []string{"openid", "email"},
			},
		},
	})

	provider, ok := registry.Get("oidc")
	if !ok {
		t.Fatal("oidc provider was not registered")
	}
	if got := provider.AuthURL("state-123"); !strings.Contains(got, "idp.example.com/oauth2/authorize") || !strings.Contains(got, "redirect_uri=https%3A%2F%2Fone-api.example.com%2Fv1%2Foauth%2Foidc%2Fcallback") {
		t.Fatalf("oidc auth url mismatch: %s", got)
	}
}

func TestSetupOAuthRegistersLarkAndWeChatWhenConfigured(t *testing.T) {
	registry := setupOAuth(&identitycfg.Config{
		OAuth: identitycfg.OAuthConfig{
			BaseURL: "https://one-api.example.com",
			Lark: identitycfg.OAuthProviderConfig{
				Enabled:      true,
				ClientID:     "lark-client",
				ClientSecret: "lark-secret",
			},
			WeChat: identitycfg.OAuthProviderConfig{
				Enabled:      true,
				ClientID:     "wechat-client",
				ClientSecret: "wechat-secret",
			},
		},
	})

	lark, ok := registry.Get("lark")
	if !ok {
		t.Fatal("lark provider was not registered")
	}
	if got := lark.AuthURL("state-123"); !strings.Contains(got, "passport.feishu.cn/suite/passport/oauth/authorize") || !strings.Contains(got, "redirect_uri=https%3A%2F%2Fone-api.example.com%2Fv1%2Foauth%2Flark%2Fcallback") {
		t.Fatalf("lark auth url mismatch: %s", got)
	}
	wechat, ok := registry.Get("wechat")
	if !ok {
		t.Fatal("wechat provider was not registered")
	}
	if got := wechat.AuthURL("state-123"); !strings.Contains(got, "open.weixin.qq.com/connect/qrconnect") || !strings.Contains(got, "redirect_uri=https%3A%2F%2Fone-api.example.com%2Fv1%2Foauth%2Fwechat%2Fcallback") {
		t.Fatalf("wechat auth url mismatch: %s", got)
	}
}
