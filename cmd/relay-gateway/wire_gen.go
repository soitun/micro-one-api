package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-kratos/kratos/v2"
	kconfig "github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"micro-one-api/api/channel/v1"
	billingv1 "micro-one-api/api/billing/v1"
	identityv1 "micro-one-api/api/identity/v1"
	appauth "micro-one-api/internal/pkg/auth"
	apptls "micro-one-api/internal/pkg/tls"
	relaybiz "micro-one-api/internal/relay/biz"
	relaycfg "micro-one-api/internal/relay/config"
	relayprovider "micro-one-api/internal/relay/provider"
	"micro-one-api/internal/relay/server"
)

func loadConfig(confPath string) (*relaycfg.Config, error) {
	source := file.NewSource(confPath)
	kratosCfg := kconfig.New(kconfig.WithSource(source))
	defer kratosCfg.Close()
	if err := kratosCfg.Load(); err != nil {
		return nil, err
	}
	var cfg relaycfg.Config
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

	tlsConfig := apptls.LoadTLSConfig()
	enableAuth := os.Getenv("ENABLE_AUTH") == "true"
	var serviceAuth *appauth.ServiceAuthConfig
	if enableAuth {
		serviceAuth, err = appauth.LoadServiceAuthConfig()
		if err != nil {
			fmt.Printf("Warning: Failed to load service auth config: %v\n", err)
			enableAuth = false
		}
	}

	providerTimeout := 30 * time.Second
	if timeoutStr := os.Getenv("RELAY_PROVIDER_TIMEOUT"); timeoutStr != "" {
		if duration, err := time.ParseDuration(timeoutStr); err == nil {
			providerTimeout = duration
		}
	}

	var identityConn, channelConn, billingConn *grpc.ClientConn
	var identityClient identityv1.IdentityServiceClient
	var channelClient channelv1.ChannelServiceClient
	var billingClient billingv1.BillingServiceClient

	if enableAuth && tlsConfig.Enabled {
		identityConn, err = createAuthenticatedClient(cfg.Clients.Identity.Endpoint, tlsConfig, serviceAuth)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create identity client: %w", err)
		}
		channelConn, err = createAuthenticatedClient(cfg.Clients.Channel.Endpoint, tlsConfig, serviceAuth)
		if err != nil {
			identityConn.Close()
			return nil, nil, fmt.Errorf("failed to create channel client: %w", err)
		}
		billingConn, err = createAuthenticatedClient(cfg.Clients.Billing.Endpoint, tlsConfig, serviceAuth)
		if err != nil {
			identityConn.Close()
			channelConn.Close()
			return nil, nil, fmt.Errorf("failed to create billing client: %w", err)
		}
	} else {
		identityConn, err = grpc.NewClient(cfg.Clients.Identity.Endpoint,
			grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return nil, nil, fmt.Errorf("failed to connect to identity: %w", err)
		}
		channelConn, err = grpc.NewClient(cfg.Clients.Channel.Endpoint,
			grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			identityConn.Close()
			return nil, nil, fmt.Errorf("failed to connect to channel: %w", err)
		}
		billingConn, err = grpc.NewClient(cfg.Clients.Billing.Endpoint,
			grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			identityConn.Close()
			channelConn.Close()
			return nil, nil, fmt.Errorf("failed to connect to billing: %w", err)
		}
	}

	identityClient = identityv1.NewIdentityServiceClient(identityConn)
	channelClient = channelv1.NewChannelServiceClient(channelConn)
	billingClient = billingv1.NewBillingServiceClient(billingConn)

	providerFactory := relayprovider.NewProviderFactory(providerTimeout)
	httpServer := server.NewHTTPServer(identityClient, channelClient, billingClient, providerFactory)
	httpServer.SetRetryConfig(
		cfg.Retry.MaxAttempts,
		cfg.Retry.InitialInterval,
		cfg.Retry.MaxInterval,
		cfg.Retry.Multiplier,
		cfg.Retry.RetryableStatus,
	)

	modelMapper, err := relaybiz.NewModelMapper(cfg.Models.Path)
	if err != nil {
		fmt.Printf("Warning: Failed to load models config: %v\n", err)
	} else {
		httpServer.SetModelMapper(modelMapper)
	}

	srv := khttp.NewServer(khttp.Address(cfg.Server.HTTP.Addr), khttp.Timeout(providerTimeout))
	httpServer.RegisterRoutes(srv)

	app := kratos.New(kratos.Name("relay-gateway"), kratos.Server(srv))

	fmt.Printf("Starting relay-gateway on %s\n", cfg.Server.HTTP.Addr)

	cleanup := func() {
		identityConn.Close()
		channelConn.Close()
		billingConn.Close()
	}

	return app, cleanup, nil
}

func createAuthenticatedClient(
	endpoint string,
	tlsConfig *apptls.TLSConfig,
	serviceAuth *appauth.ServiceAuthConfig,
) (*grpc.ClientConn, error) {
	creds, err := apptls.CreateClientCredentials(tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS credentials: %w", err)
	}
	opts := []grpc.DialOption{grpc.WithTransportCredentials(creds)}
	if serviceAuth != nil && serviceAuth.Token != "" {
		tokenCreds := &tokenAuth{token: serviceAuth.Token}
		opts = append(opts, grpc.WithPerRPCCredentials(tokenCreds))
	}
	conn, err := grpc.NewClient(endpoint, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", endpoint, err)
	}
	return conn, nil
}

type tokenAuth struct {
	token string
}

func (t *tokenAuth) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": "Bearer " + t.token,
	}, nil
}

func (t *tokenAuth) RequireTransportSecurity() bool {
	return true
}
