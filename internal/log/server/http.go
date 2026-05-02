package server

import (
	"net/http"

	khttp "github.com/go-kratos/kratos/v2/transport/http"

	"micro-one-api/internal/log/service"
)

// NewHTTPServer wires HTTP transport for log-service.
func NewHTTPServer(addr string, svc *service.LogService) *khttp.Server {
	srv := khttp.NewServer(
		khttp.Address(addr),
	)
	srv.HandleFunc("/v1/logs", func(w http.ResponseWriter, r *http.Request) {
		// GET /v1/logs (list) or POST /v1/logs (ingest)
		switch r.Method {
		case http.MethodGet:
			svc.ListLogs(w, r)
		case http.MethodPost:
			svc.IngestLog(w, r)
		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})
	srv.HandleFunc("/v1/logs/", func(w http.ResponseWriter, r *http.Request) {
		// GET /v1/logs/{id}
		svc.GetLog(w, r)
	})
	srv.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})
	return srv
}
