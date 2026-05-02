package server

import (
	"micro-one-api/internal/config/service"

	kgrpc "github.com/go-kratos/kratos/v2/transport/grpc"
)

// NewGRPCServer wires gRPC transport for config-service.
func NewGRPCServer(addr string, svc *service.ConfigService) *kgrpc.Server {
	srv := kgrpc.NewServer(
		kgrpc.Address(addr),
	)
	// Register gRPC service handlers here when proto is defined.
	_ = svc
	return srv
}
