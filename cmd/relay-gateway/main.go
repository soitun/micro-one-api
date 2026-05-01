package main

import (
	"fmt"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"micro-one-api/api/channel/v1"
	identityv1 "micro-one-api/api/identity/v1"
	"micro-one-api/internal/relay/provider"
	"micro-one-api/internal/relay/server"

	"github.com/go-kratos/kratos/v2"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

func main() {
	identityEndpoint := os.Getenv("IDENTITY_GRPC_ENDPOINT")
	if identityEndpoint == "" {
		identityEndpoint = "127.0.0.1:9001"
	}
	channelEndpoint := os.Getenv("CHANNEL_GRPC_ENDPOINT")
	if channelEndpoint == "" {
		channelEndpoint = "127.0.0.1:9002"
	}
	httpAddr := os.Getenv("RELAY_HTTP_ADDR")
	if httpAddr == "" {
		httpAddr = ":8080"
	}

	identityConn, err := grpc.Dial(identityEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(fmt.Sprintf("failed to connect to identity service: %v", err))
	}
	defer identityConn.Close()

	channelConn, err := grpc.Dial(channelEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(fmt.Sprintf("failed to connect to channel service: %v", err))
	}
	defer channelConn.Close()

	identityClient := identityv1.NewIdentityServiceClient(identityConn)
	channelClient := channelv1.NewChannelServiceClient(channelConn)

	providerTimeout := 30 * time.Second
	if timeoutStr := os.Getenv("RELAY_PROVIDER_TIMEOUT"); timeoutStr != "" {
		if duration, err := time.ParseDuration(timeoutStr); err == nil {
			providerTimeout = duration
		}
	}

	providerFactory := provider.NewProviderFactory(providerTimeout)

	httpServer := server.NewHTTPServer(identityClient, channelClient, providerFactory)

	srv := khttp.NewServer(
		khttp.Address(httpAddr),
		khttp.Timeout(providerTimeout),
	)
	httpServer.RegisterRoutes(srv)

	app := kratos.New(
		kratos.Name("relay-gateway"),
		kratos.Server(srv),
	)

	if err := app.Run(); err != nil {
		panic(fmt.Sprintf("failed to start relay gateway: %v", err))
	}
}
