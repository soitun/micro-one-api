package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	adminv1 "micro-one-api/api/admin/v1"
	billingv1 "micro-one-api/api/billing/v1"
	"micro-one-api/internal/admin/service"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	grpcx "github.com/go-kratos/kratos/v2/transport/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// 配置
	grpcAddr := os.Getenv("ADMIN_GRPC_ADDR")
	if grpcAddr == "" {
		grpcAddr = ":9005"
	}

	billingServiceAddr := os.Getenv("BILLING_SERVICE_ADDR")
	if billingServiceAddr == "" {
		billingServiceAddr = "localhost:9004"
	}

	// 连接到 Billing Service
	billingConn, err := grpc.Dial(billingServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		panic(fmt.Sprintf("failed to connect to billing service: %v", err))
	}
	defer billingConn.Close()

	billingClient := billingv1.NewBillingServiceClient(billingConn)

	// 创建 Admin Service
	adminService := service.NewAdminService(billingClient)

	// 创建 Kratos gRPC Server
	grpcSrv := grpcx.NewServer(
		grpcx.Address(grpcAddr),
		grpcx.Middleware(
			recovery.Recovery(),
		),
	)
	adminv1.RegisterAdminServiceServer(grpcSrv, adminService)

	// 创建 Kratos 应用
	app := kratos.New(
		kratos.Name("admin-api"),
		kratos.Server(grpcSrv),
	)

	// 优雅关闭
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		fmt.Println("Shutting down server...")
		app.Stop()
	}()

	// 运行应用
	if err := app.Run(); err != nil {
		panic(err)
	}
}