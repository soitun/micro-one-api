//go:build wireinject
// +build wireinject

package main

import (
	"github.com/go-kratos/kratos/v2"
	"github.com/google/wire"

	admincfg "micro-one-api/internal/admin/config"
	"micro-one-api/internal/admin/service"
)

var ProviderSet = wire.NewSet(
	service.NewAdminService,
)

func InitApp(confPath string) (*admincfg.Config, *kratos.App, func(), error) {
	panic(wire.Build(
		loadConfig,
		newClients,
		ProviderSet,
		newApp,
	))
}

func newApp(cfg *admincfg.Config, svc *service.AdminService) (*kratos.App, func()) {
	grpcSrv := newGRPCServer(cfg, svc)
	app := kratos.New(
		kratos.Name("admin-api"),
		kratos.Server(grpcSrv),
	)
	return app, func() {}
}
