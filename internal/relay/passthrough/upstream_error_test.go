package passthrough

import (
	"net/http"
	"testing"
)

func TestClassify(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		kind   Kind
	}{
		{name: "429 passthrough", status: http.StatusTooManyRequests, kind: KindPassthrough},
		{name: "401 passthrough", status: http.StatusUnauthorized, kind: KindPassthrough},
		{name: "403 passthrough", status: http.StatusForbidden, kind: KindPassthrough},
		{name: "cyber policy", status: http.StatusBadRequest, body: `{"error":"cyber_policy"}`, kind: KindCyberBlocked},
		{name: "5xx retryable", status: http.StatusBadGateway, kind: KindRetryable},
		{name: "other 4xx non retryable", status: http.StatusBadRequest, kind: KindNonRetryable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Classify(tt.status, []byte(tt.body))
			if got.Kind != tt.kind {
				t.Fatalf("kind = %s, want %s", got.Kind, tt.kind)
			}
		})
	}
}
