package metrics

import (
	"net/http"
	"strconv"
	"time"
)

// responseWriter wraps http.ResponseWriter to capture status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// HTTPMiddleware returns an HTTP middleware that records request metrics.
func HTTPMiddleware(service string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ActiveRequests.WithLabelValues(service).Inc()
			defer ActiveRequests.WithLabelValues(service).Dec()

			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(rw, r)

			duration := time.Since(start).Seconds()
			status := strconv.Itoa(rw.statusCode)

			HTTPRequestTotal.WithLabelValues(service, r.Method, r.URL.Path, status).Inc()
			HTTPRequestDuration.WithLabelValues(service, r.Method, r.URL.Path).Observe(duration)
		})
	}
}
