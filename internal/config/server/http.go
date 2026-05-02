package server

import (
	"net/http"

	khttp "github.com/go-kratos/kratos/v2/transport/http"

	"micro-one-api/internal/config/service"
)

// NewHTTPServer wires HTTP transport for config-service.
func NewHTTPServer(addr string, svc *service.ConfigService) *khttp.Server {
	srv := khttp.NewServer(
		khttp.Address(addr),
	)
	srv.HandleFunc("/v1/configs/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			// Distinguish between /v1/configs/{ns} and /v1/configs/{ns}/{key}
			rest := r.URL.Path[len("/v1/configs/"):]
			if countSlashes(rest) >= 1 {
				svc.GetConfig(w, r)
			} else {
				svc.ListConfigs(w, r)
			}
		case http.MethodPut:
			svc.SetConfig(w, r)
		case http.MethodDelete:
			svc.DeleteConfig(w, r)
		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})
	srv.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})
	return srv
}

// countSlashes counts the number of '/' characters in s.
func countSlashes(s string) int {
	n := 0
	for _, c := range s {
		if c == '/' {
			n++
		}
	}
	return n
}
