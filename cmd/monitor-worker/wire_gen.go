//go:build !wireinject

package main

import (
	"github.com/go-kratos/kratos/v2"
	kconfig "github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"

	monitorcfg "micro-one-api/internal/monitor/config"
	"micro-one-api/internal/monitor/biz"
	"micro-one-api/internal/monitor/data"
	"micro-one-api/internal/monitor/server"
	"micro-one-api/internal/monitor/service"
)

func loadConfig(confPath string) (*monitorcfg.Config, error) {
	source := file.NewSource(confPath)
	kratosCfg := kconfig.New(kconfig.WithSource(source))
	defer kratosCfg.Close()
	if err := kratosCfg.Load(); err != nil {
		return nil, err
	}
	var cfg monitorcfg.Config
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

	uc := biz.NewMonitorUsecase(repo)
	svc := service.NewMonitorService(uc)
	grpcSrv := server.NewGRPCServer(cfg.Server.GRPC.Addr, svc)
	httpSrv := server.NewHTTPServer(cfg.Server.HTTP.Addr, svc)

	app := kratos.New(
		kratos.Name("monitor-worker"),
		kratos.Server(grpcSrv, httpSrv),
	)

	return app, func() {}, nil
}
