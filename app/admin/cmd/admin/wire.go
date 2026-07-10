//go:build wireinject
// +build wireinject

package main

import (
	"github.com/go-kratos/kratos/v2"
	"github.com/google/wire"

	admincfg "micro-one-api/app/admin/internal/conf"
	"micro-one-api/app/admin/internal/server"
	"micro-one-api/app/admin/internal/service"
)

var ProviderSet = wire.NewSet(
	service.NewAdminService,
)

func InitApp(confPath string) (*kratos.App, func(), error) {
	panic(wire.Build(
		loadConfig,
		newClients,
		ProviderSet,
		newApp,
	))
}

func newApp(cfg *admincfg.Config, svc *service.AdminService) (*kratos.App, func()) {
	grpcSrv := newGRPCServer(cfg, svc)
	httpSrv := server.NewHTTPServer(cfg.Server.HTTP.Addr, svc, cfg.Clients.Identity.HTTPEndpoint, cfg.Server.HTTP.WebRoot)
	app := kratos.New(
		kratos.Name("admin-api"),
		kratos.Server(grpcSrv, httpSrv),
	)
	return app, func() {}
}
