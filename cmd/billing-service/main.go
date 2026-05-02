package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"micro-one-api/internal/billing/biz"
	"micro-one-api/internal/billing/data"
	"micro-one-api/internal/billing/server"
	"micro-one-api/internal/billing/service"

	"github.com/go-kratos/kratos/v2"
)

func main() {
	grpcAddr := os.Getenv("BILLING_GRPC_ADDR")
	if grpcAddr == "" {
		grpcAddr = ":9004"
	}

	httpAddr := os.Getenv("BILLING_HTTP_ADDR")
	if httpAddr == "" {
		httpAddr = ":8004"
	}

	d, err := data.NewData()
	if err != nil {
		panic(err)
	}

	uc := biz.NewBillingUsecase(
		d.AccountRepo(),
		d.ReservationRepo(),
		d.LedgerRepo(),
		d.RedeemRepo(),
	)
	svc := service.NewBillingService(uc)

	grpcSrv := server.NewGRPCServer(grpcAddr, svc)
	httpSrv := server.NewHTTPServer(httpAddr, svc)

	app := kratos.New(
		kratos.Name("billing-service"),
		kratos.Server(grpcSrv, httpSrv),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cleanupJob := biz.NewCleanupJob(uc, 1*time.Minute)
	go cleanupJob.Start(ctx)

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		cancel()
		cleanupJob.Stop()
	}()

	if err := app.Run(); err != nil {
		panic(err)
	}
}
