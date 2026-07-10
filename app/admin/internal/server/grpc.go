package server

import (
	adminv1 "micro-one-api/api/admin/v1"
	"micro-one-api/app/admin/internal/service"

	"google.golang.org/grpc"
)

// NewGRPCServer creates a gRPC server and registers the AdminService.
func NewGRPCServer(svc *service.AdminService) *grpc.Server {
	srv := grpc.NewServer()
	adminv1.RegisterAdminServiceServer(srv, svc)
	return srv
}
