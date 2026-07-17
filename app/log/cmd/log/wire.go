//go:build wireinject
// +build wireinject

package main

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/google/wire"

	"micro-one-api/app/log/internal/biz"
	logcfg "micro-one-api/app/log/internal/conf"
	"micro-one-api/app/log/internal/data"
	"micro-one-api/app/log/internal/server"
	"micro-one-api/app/log/internal/service"

	applogger "micro-one-api/platform/logging"
	appregistry "micro-one-api/platform/registry"

	"go.uber.org/zap"
)

var ProviderSet = wire.NewSet(
	newRepo,
	biz.NewLogUsecase,
	service.NewLogService,
	server.NewGRPCServer,
	server.NewHTTPServer,
	provideRegistrar,
	wire.Bind(new(biz.LogRepo), new(*data.Repository)),
)

func newRepo(cfg *logcfg.Config) (*data.Repository, error) {
	return data.NewRepositoryFromEnv(cfg.Data.Database.Driver, cfg.Data.Database.Source)
}

type registrarResult struct {
	Registrar registry.Registrar
}

func provideRegistrar(cfg *logcfg.Config) registrarResult {
	registrar, err := appregistry.NewRegistrar(cfg.Registry)
	if err != nil {
		return registrarResult{}
	}
	return registrarResult{Registrar: registrar}
}

func InitApp(confPath string) (*kratos.App, func(), error) {
	panic(wire.Build(
		loadConfig,
		ProviderSet,
		newApp,
	))
}

func newApp(cfg *logcfg.Config, repo *data.Repository, uc *biz.LogUsecase, svc *service.LogService, reg registrarResult) (*kratos.App, func()) {
	grpcSrv := server.NewGRPCServer(cfg.Server.GRPC.Addr, svc)
	httpSrv := server.NewHTTPServer(cfg.Server.HTTP.Addr, svc)
	cleanupRetention := startLogRetentionCleanup(uc, cfg.Log.RetentionDays)
	partitionCtx, partitionCancel := context.WithCancel(context.Background())
	partitionStop := startPartitionMaintenance(partitionCtx, repo.DB(), cfg.Partition)

	// Phase 2.3: optional async batch log writer. When disabled (default),
	// LogUsecase.IngestLog falls back to synchronous repo.Create; when
	// enabled, entries are queued and flushed in batches via
	// Repository.CreateBatch (gorm CreateInBatches), lowering per-request
	// log latency. The writer's lifecycle is tied to the app cleanup.
	var batchWriter *biz.BatchLogWriter
	var batchStop func()
	if cfg.Log.BatchEnabled {
		batchSize := cfg.Log.BatchSize
		if batchSize <= 0 {
			batchSize = 100
		}
		flushInterval := parseLogFlushInterval(cfg.Log.BatchFlushInterval, time.Second)
		batchWriter = biz.NewBatchLogWriter(repo, batchSize, flushInterval)
		batchWriter.Start()
		uc.SetBatchWriter(batchWriter)
		batchStop = batchWriter.Stop
		if applogger.Log != nil {
			applogger.Log.Info("log batch writer enabled",
				zap.Int("batch_size", batchSize),
				zap.Duration("flush_interval", flushInterval))
		}
	}

	opts := []kratos.Option{
		kratos.Name("log-service"),
		kratos.Server(grpcSrv, httpSrv),
	}
	if reg.Registrar != nil {
		opts = append(opts, kratos.Registrar(reg.Registrar))
	}
	app := kratos.New(opts...)
	return app, func() {
		cleanupRetention()
		partitionCancel()
		if partitionStop != nil {
			partitionStop()
		}
		if batchStop != nil {
			batchStop()
		}
	}
}

// parseLogFlushInterval parses a duration string with a fallback. Empty or
// unparsable values fall back to the provided default so a missing config
// field never blocks the writer.
func parseLogFlushInterval(raw string, fallback time.Duration) time.Duration {
	if raw == "" {
		return fallback
	}
	if d, err := time.ParseDuration(raw); err == nil && d > 0 {
		return d
	}
	return fallback
}
