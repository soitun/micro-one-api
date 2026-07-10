//go:build wireinject
// +build wireinject

package main

import (
	"github.com/go-kratos/kratos/v2"
	"github.com/google/wire"

	relaybiz "micro-one-api/app/relay/interface/internal/biz"
	relaycfg "micro-one-api/app/relay/interface/internal/config"
	relayprovider "micro-one-api/domain/upstream/provider"
	"micro-one-api/app/relay/interface/internal/server"
)

var ProviderSet = wire.NewSet(
	relayprovider.NewProviderFactory,
	relaybiz.NewRelayUsecase,
	server.NewHTTPServer,
)

func InitApp(confPath string) (*kratos.App, func(), error) {
	panic(wire.Build(
		loadConfig,
		newClients,
		newModelMapper,
		newRetryPolicy,
		ProviderSet,
		newApp,
	))
}

func newApp(cfg *relaycfg.Config, httpServer *server.HTTPServer) (*kratos.App, func()) {
	srv := newHTTPServer(cfg, httpServer)
	app := kratos.New(
		kratos.Name("relay-gateway"),
		kratos.Server(srv),
	)
	return app, func() {}
}
