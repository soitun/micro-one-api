//go:build wireinject
// +build wireinject

package main

import (
	"github.com/go-kratos/kratos/v2"
	"github.com/google/wire"

	relaycfg "micro-one-api/internal/relay/config"
	relayprovider "micro-one-api/internal/relay/provider"
	"micro-one-api/internal/relay/server"
)

var ProviderSet = wire.NewSet(
	relayprovider.NewProviderFactory,
	server.NewHTTPServer,
)

func InitApp(confPath string) (*relaycfg.Config, *kratos.App, func(), error) {
	panic(wire.Build(
		loadConfig,
		newClients,
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
