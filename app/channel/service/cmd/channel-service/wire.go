//go:build wireinject
// +build wireinject

package main

import (
	"github.com/go-kratos/kratos/v2"
	"github.com/google/wire"

	"micro-one-api/app/channel/service/internal/biz"
	channelcfg "micro-one-api/app/channel/service/internal/config"
	"micro-one-api/app/channel/service/internal/data"
	"micro-one-api/app/channel/service/internal/server"
	"micro-one-api/app/channel/service/internal/service"
)

var ProviderSet = wire.NewSet(
	data.NewRepositoryFromEnv,
	biz.NewChannelUsecase,
	service.NewChannelService,
	server.NewGRPCServer,
	server.NewHTTPServer,
)

func InitApp(confPath string) (*kratos.App, func(), error) {
	panic(wire.Build(
		loadConfig,
		ProviderSet,
		newApp,
	))
}

func newApp(cfg *channelcfg.Config, svc *service.ChannelService) (*kratos.App, func()) {
	grpcSrv := server.NewGRPCServer(cfg.Server.GRPC.Addr, svc)
	httpSrv := server.NewHTTPServer(cfg.Server.HTTP.Addr, svc.Usecase())
	app := kratos.New(
		kratos.Name("channel-service"),
		kratos.Server(grpcSrv, httpSrv),
	)
	return app, func() {}
}
