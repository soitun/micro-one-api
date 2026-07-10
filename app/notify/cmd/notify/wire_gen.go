//go:build !wireinject

package main

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2"
	kconfig "github.com/go-kratos/kratos/v2/config"
	"go.uber.org/zap"

	"micro-one-api/app/notify/internal/biz"
	notifycfg "micro-one-api/app/notify/internal/conf"
	"micro-one-api/app/notify/internal/data"
	"micro-one-api/app/notify/internal/server"
	"micro-one-api/app/notify/internal/service"
	applogger "micro-one-api/platform/logging"
	appregistry "micro-one-api/platform/registry"
	"micro-one-api/platform/config"
)

func loadConfig(confPath string) (*notifycfg.Config, error) {
	source := xconfig.NewEnvFileSource(confPath)
	kratosCfg := kconfig.New(kconfig.WithSource(source), kconfig.WithResolveActualTypes(true))
	defer kratosCfg.Close()
	if err := kratosCfg.Load(); err != nil {
		return nil, err
	}
	var cfg notifycfg.Config
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

	uc := biz.NewNotifyUsecase(repo)
	svc := service.NewNotifyService(uc)
	grpcSrv := server.NewGRPCServer(cfg.Server.GRPC.Addr, svc)
	httpSrv := server.NewHTTPServer(cfg.Server.HTTP.Addr, svc)
	dispatchInterval, err := time.ParseDuration(cfg.Notify.DispatchInterval)
	if err != nil {
		dispatchInterval = 30 * time.Second
	}
	sender := biz.NewMultiSender(biz.SenderConfig{
		WebhookURL:        cfg.Notify.WebhookURL,
		SMTPHost:          cfg.Notify.SMTPHost,
		SMTPPort:          cfg.Notify.SMTPPort,
		SMTPUser:          cfg.Notify.SMTPUser,
		SMTPPass:          cfg.Notify.SMTPPass,
		SMTPFrom:          cfg.Notify.SMTPFrom,
		WeComWebhookURL:   cfg.Notify.WeComWebhookURL,
		DingTalkWebhookURL: cfg.Notify.DingTalkWebhookURL,
		FeishuWebhookURL:  cfg.Notify.FeishuWebhookURL,
		SlackWebhookURL:   cfg.Notify.SlackWebhookURL,
	})
	dispatcher := biz.NewDispatcher(uc, sender, dispatchInterval, cfg.Notify.DispatchBatch, cfg.Notify.MaxRetry)
	stopDispatcher := dispatcher.Start(context.Background())

	registrar, rErr := appregistry.NewRegistrar(cfg.Registry)
	if rErr != nil {
		applogger.Log.Warn("failed to create registrar", zap.Error(rErr))
	}

	kratosOpts := []kratos.Option{
		kratos.Name("notify-worker"),
		kratos.Server(grpcSrv, httpSrv),
	}
	if registrar != nil {
		kratosOpts = append(kratosOpts, kratos.Registrar(registrar))
	}

	app := kratos.New(kratosOpts...)

	return app, stopDispatcher, nil
}
