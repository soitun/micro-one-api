package server

import (
	billingv1 "micro-one-api/api/billing/v1"
	"micro-one-api/internal/billing/service"

	kgrpc "github.com/go-kratos/kratos/v2/transport/grpc"
)

// NewGRPCServer wires gRPC transport for billing-service.
func NewGRPCServer(addr string, svc *service.BillingService) *kgrpc.Server {
	srv := kgrpc.NewServer(
		kgrpc.Address(addr),
	)
	billingv1.RegisterBillingServiceServer(srv, svc)
	return srv
}
