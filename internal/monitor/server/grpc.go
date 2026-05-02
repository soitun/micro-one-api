package server

import (
	"micro-one-api/internal/monitor/service"

	kgrpc "github.com/go-kratos/kratos/v2/transport/grpc"
)

// NewGRPCServer wires gRPC transport for monitor-worker.
func NewGRPCServer(addr string, svc *service.MonitorService) *kgrpc.Server {
	srv := kgrpc.NewServer(
		kgrpc.Address(addr),
	)
	// Register gRPC service handlers here when proto is defined.
	_ = svc
	return srv
}
