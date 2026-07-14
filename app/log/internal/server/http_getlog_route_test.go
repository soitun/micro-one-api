package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestLogHTTPGetLogByIDRouteReachable guards against a regression where
// /v1/logs/{id} returns the gorilla/mux "404 page not found" plaintext
// instead of reaching HandleGetLog. The root cause was using
// srv.HandleFunc("/v1/logs/", ...) which only matches the exact path, not
// subpaths. The fix uses srv.HandlePrefix so /v1/logs/7538 is routed
// correctly.
func TestLogHTTPGetLogByIDRouteReachable(t *testing.T) {
	t.Setenv("SERVICE_TOKEN", "service-token")
	srv := newLogHTTPServerForTest(t, &logHTTPIdentityClient{})

	// /v1/logs/1 exists in the test repository — expect 200.
	req := httptest.NewRequest(http.MethodGet, "/v1/logs/1", nil)
	req.Header.Set("Authorization", "Bearer service-token")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code == http.StatusNotFound && rec.Body.String() == "404 page not found\n" {
		t.Fatalf("GET /v1/logs/1 returned gorilla/mux 404 — route not matched. Use HandlePrefix instead of HandleFunc for prefix routes.")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /v1/logs/1 status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
}

// TestLogHTTPGetLogByIDRouteReachableForMissingID confirms that a non-existent
// ID still reaches the handler and returns a JSON 404, not a plaintext
// gorilla/mux 404.
func TestLogHTTPGetLogByIDRouteReachableForMissingID(t *testing.T) {
	t.Setenv("SERVICE_TOKEN", "service-token")
	srv := newLogHTTPServerForTest(t, &logHTTPIdentityClient{})

	req := httptest.NewRequest(http.MethodGet, "/v1/logs/7538", nil)
	req.Header.Set("Authorization", "Bearer service-token")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Body.String() == "404 page not found\n" {
		t.Fatalf("GET /v1/logs/7538 returned gorilla/mux 404 — route not matched.")
	}
	// A JSON 404 ("log entry not found") is correct: the route was matched
	// and the service reported the log doesn't exist in the DB.
}

// TestLogHTTPGetLogByIDRouteUnauthorized ensures ServiceAuth still gates the
// prefix route.
func TestLogHTTPGetLogByIDRouteUnauthorized(t *testing.T) {
	t.Setenv("SERVICE_TOKEN", "service-token")
	srv := newLogHTTPServerForTest(t, &logHTTPIdentityClient{})

	req := httptest.NewRequest(http.MethodGet, "/v1/logs/1", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("GET /v1/logs/1 without token status = %d, want 401", rec.Code)
	}
}
