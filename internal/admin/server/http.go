package server

import (
	"net/http"

	"micro-one-api/internal/pkg/metrics"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

// NewHTTPServer wires HTTP transport for admin-api.
func NewHTTPServer(addr string) *khttp.Server {
	srv := khttp.NewServer(
		khttp.Address(addr),
	)
	srv.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		metrics.Handler().ServeHTTP(w, r)
	})
	srv.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})
	return srv
}
