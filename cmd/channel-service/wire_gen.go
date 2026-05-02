//go:build !wireinject

package main

import (
	"github.com/go-kratos/kratos/v2"
	kconfig "github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"

	"micro-one-api/internal/channel/biz"
	channelcfg "micro-one-api/internal/channel/config"
	"micro-one-api/internal/channel/data"
	"micro-one-api/internal/channel/server"
	"micro-one-api/internal/channel/service"
)

func loadConfig(confPath string) (*channelcfg.Config, error) {
	source := file.NewSource(confPath)
	kratosCfg := kconfig.New(kconfig.WithSource(source))
	defer kratosCfg.Close()
	if err := kratosCfg.Load(); err != nil {
		return nil, err
	}
	var cfg channelcfg.Config
	if err := kratosCfg.Scan(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// InitApp loads config and builds the Kratos application.
func InitApp(confPath string) (*kratos.App, func(), error) {
	cfg, err := loadConfig(confPath)
	if err != nil {
		return nil, nil, err
	}

	repo, err := data.NewRepositoryFromEnv(cfg.Data.Database.Source)
	if err != nil {
		return nil, nil, err
	}

	uc := biz.NewChannelUsecase(repo)
	svc := service.NewChannelService(uc)
	grpcSrv := server.NewGRPCServer(cfg.Server.GRPC.Addr, svc)

	app := kratos.New(
		kratos.Name("channel-service"),
		kratos.Server(grpcSrv),
	)

	return app, func() {}, nil
}
