//go:build !wireinject

package main

import (
	"github.com/go-kratos/kratos/v2"
	kconfig "github.com/go-kratos/kratos/v2/config"
	"go.uber.org/zap"

	"micro-one-api/app/config/service/internal/biz"
	configcfg "micro-one-api/app/config/service/internal/config"
	"micro-one-api/app/config/service/internal/data"
	"micro-one-api/app/config/service/internal/server"
	"micro-one-api/app/config/service/internal/service"
	"micro-one-api/platform/events"
	applogger "micro-one-api/platform/logging"
	appregistry "micro-one-api/platform/registry"
	"micro-one-api/platform/config"
)

func loadConfig(confPath string) (*configcfg.Config, error) {
	source := xconfig.NewEnvFileSource(confPath)
	kratosCfg := kconfig.New(kconfig.WithSource(source), kconfig.WithResolveActualTypes(true))
	defer kratosCfg.Close()
	if err := kratosCfg.Load(); err != nil {
		return nil, err
	}
	var cfg configcfg.Config
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

	repo, err := data.NewRepositoryFromEnv(cfg.Data.Database.Driver, cfg.Data.Database.Source)
	if err != nil {
		return nil, nil, err
	}

	eventBus := events.NewConfiguredEventBus(repo.Redis(), "config-service")
	uc := biz.NewConfigUsecase(repo, eventBus)
	svc := service.NewConfigService(uc)
	grpcSrv := server.NewGRPCServer(cfg.Server.GRPC.Addr, svc)
	httpSrv := server.NewHTTPServer(cfg.Server.HTTP.Addr, svc)

	registrar, rErr := appregistry.NewRegistrar(cfg.Registry)
	if rErr != nil {
		applogger.Log.Warn("failed to create registrar", zap.Error(rErr))
	}

	kratosOpts := []kratos.Option{
		kratos.Name("config-service"),
		kratos.Server(grpcSrv, httpSrv),
	}
	if registrar != nil {
		kratosOpts = append(kratosOpts, kratos.Registrar(registrar))
	}

	app := kratos.New(kratosOpts...)

	return app, func() {}, nil
}
