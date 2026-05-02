//go:build wireinject
// +build wireinject

package main

import (
	"github.com/go-kratos/kratos/v2"
	"github.com/google/wire"

	logcfg "micro-one-api/internal/log/config"
	"micro-one-api/internal/log/biz"
	"micro-one-api/internal/log/data"
	"micro-one-api/internal/log/server"
	"micro-one-api/internal/log/service"
)

var ProviderSet = wire.NewSet(
	data.NewRepositoryFromEnv,
	biz.NewLogUsecase,
	service.NewLogService,
	server.NewGRPCServer,
)

func InitApp(confPath string) (*kratos.App, func(), error) {
	panic(wire.Build(
		loadConfig,
		ProviderSet,
		newApp,
	))
}

func newApp(cfg *logcfg.Config, svc *service.LogService) (*kratos.App, func()) {
	grpcSrv := server.NewGRPCServer(cfg.Server.GRPC.Addr, svc)
	httpSrv := server.NewHTTPServer(cfg.Server.HTTP.Addr, svc)
	app := kratos.New(
		kratos.Name("log-service"),
		kratos.Server(grpcSrv, httpSrv),
	)
	return app, func() {}
}
