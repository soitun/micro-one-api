//go:build wireinject
// +build wireinject

package main

import (
	"github.com/go-kratos/kratos/v2"
	"github.com/google/wire"

	monitorcfg "micro-one-api/internal/monitor/config"
	"micro-one-api/internal/monitor/biz"
	"micro-one-api/internal/monitor/data"
	"micro-one-api/internal/monitor/server"
	"micro-one-api/internal/monitor/service"
)

var ProviderSet = wire.NewSet(
	data.NewRepositoryFromEnv,
	biz.NewMonitorUsecase,
	service.NewMonitorService,
	server.NewGRPCServer,
)

func InitApp(confPath string) (*kratos.App, func(), error) {
	panic(wire.Build(
		loadConfig,
		ProviderSet,
		newApp,
	))
}

func newApp(cfg *monitorcfg.Config, svc *service.MonitorService) (*kratos.App, func()) {
	grpcSrv := server.NewGRPCServer(cfg.Server.GRPC.Addr, svc)
	httpSrv := server.NewHTTPServer(cfg.Server.HTTP.Addr, svc)
	app := kratos.New(
		kratos.Name("monitor-worker"),
		kratos.Server(grpcSrv, httpSrv),
	)
	return app, func() {}
}
