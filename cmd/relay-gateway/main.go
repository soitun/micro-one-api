package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	identityv1 "micro-one-api/api/identity/v1"
	channelv1 "micro-one-api/api/channel/v1"
	"micro-one-api/internal/relay/provider"
	"micro-one-api/internal/relay/server"
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

	mux := http.NewServeMux()
	httpServer.RegisterRoutes(mux)

	srv := &http.Server{
		Addr:    httpAddr,
		Handler: mux,
	}

	log.Printf("Relay Gateway HTTP server listening on %s", httpAddr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		panic(fmt.Sprintf("failed to start HTTP server: %v", err))
	}
}
