//go:build wireinject
// +build wireinject

package main

import (
	"github.com/go-kratos/kratos/v2"
	"github.com/google/wire"

	notifycfg "micro-one-api/internal/notify/config"
	"micro-one-api/internal/notify/biz"
	"micro-one-api/internal/notify/data"
	"micro-one-api/internal/notify/server"
	"micro-one-api/internal/notify/service"
)

var ProviderSet = wire.NewSet(
	data.NewRepositoryFromEnv,
	biz.NewNotifyUsecase,
	service.NewNotifyService,
	server.NewGRPCServer,
)

func InitApp(confPath string) (*kratos.App, func(), error) {
	panic(wire.Build(
		loadConfig,
		ProviderSet,
		newApp,
	))
}

func newApp(cfg *notifycfg.Config, svc *service.NotifyService) (*kratos.App, func()) {
	grpcSrv := server.NewGRPCServer(cfg.Server.GRPC.Addr, svc)
	httpSrv := server.NewHTTPServer(cfg.Server.HTTP.Addr, svc)
	app := kratos.New(
		kratos.Name("notify-worker"),
		kratos.Server(grpcSrv, httpSrv),
	)
	return app, func() {}
}
