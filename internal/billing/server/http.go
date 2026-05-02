package server

import (
	"micro-one-api/internal/billing/service"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

// NewHTTPServer wires HTTP transport for billing-service.
func NewHTTPServer(addr string, svc *service.BillingService) *khttp.Server {
	srv := khttp.NewServer(
		khttp.Address(addr),
	)
	return srv
}
