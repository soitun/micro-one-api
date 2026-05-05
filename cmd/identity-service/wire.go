//go:build wireinject
// +build wireinject

package main

import (
	"github.com/go-kratos/kratos/v2"
	"github.com/google/wire"

	identitycfg "micro-one-api/internal/identity/config"
	"micro-one-api/internal/identity/biz"
	"micro-one-api/internal/identity/data"
	"micro-one-api/internal/identity/server"
	"micro-one-api/internal/identity/service"
	"micro-one-api/internal/pkg/oauth"
)

var ProviderSet = wire.NewSet(
	data.NewRepositoryFromEnv,
	biz.NewIdentityUsecase,
	service.NewIdentityService,
	server.NewGRPCServer,
	server.NewHTTPServer,
)

func InitApp(confPath string) (*kratos.App, func(), error) {
	panic(wire.Build(
		loadConfig,
		ProviderSet,
		setupOAuth,
		newApp,
	))
}

func newApp(cfg *identitycfg.Config, svc *service.IdentityService, uc *biz.IdentityUsecase, oauthRegistry *oauth.ProviderRegistry) (*kratos.App, func()) {
	grpcSrv := server.NewGRPCServer(cfg.Server.GRPC.Addr, svc)
	httpSrv := server.NewHTTPServer(cfg.Server.HTTP.Addr, uc, oauthRegistry)
	app := kratos.New(
		kratos.Name("identity-service"),
		kratos.Server(grpcSrv, httpSrv),
	)
	return app, func() {}
}
