package server

import (
	"net/http"

	"micro-one-api/internal/billing/service"
	"micro-one-api/internal/pkg/metrics"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

// NewHTTPServer wires HTTP transport for billing-service.
func NewHTTPServer(addr string, svc *service.BillingService) *khttp.Server {
	srv := khttp.NewServer(
		khttp.Address(addr),
	)
	srv.HandleFunc("/v1/reconciliation", svc.HandleReconciliation)
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
