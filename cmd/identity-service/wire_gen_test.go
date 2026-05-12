package main

import (
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
