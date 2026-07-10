package tls

import "testing"

func TestAllowInsecureSkipVerifyFromEnvRequiresExplicitGate(t *testing.T) {
	t.Setenv("TLS_INSECURE_SKIP_VERIFY", "true")
	t.Setenv("TLS_ALLOW_INSECURE_SKIP_VERIFY", "")
	if allowInsecureSkipVerifyFromEnv() {
		t.Fatal("skip verify enabled without allow gate")
	}

	t.Setenv("TLS_ALLOW_INSECURE_SKIP_VERIFY", "true")
	if !allowInsecureSkipVerifyFromEnv() {
		t.Fatal("skip verify disabled with both gates enabled")
	}
}
