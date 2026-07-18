//go:build wireinject
// +build wireinject

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-kratos/kratos/v2"
	"github.com/google/wire"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	billingv1 "micro-one-api/api/billing/v1"
	"micro-one-api/api/channel/v1"
	identityv1 "micro-one-api/api/identity/v1"
	logv1 "micro-one-api/api/log/v1"
	relayprovider "micro-one-api/domain/upstream/provider"
	relaybiz "micro-one-api/internal/biz"
	relaycfg "micro-one-api/internal/conf"
	relaydata "micro-one-api/internal/data"
	relayidentity "micro-one-api/internal/identity"
	"micro-one-api/internal/server"
	relayservice "micro-one-api/internal/service"
	apptimeout "micro-one-api/pkg/timeout"
	appaudit "micro-one-api/platform/audit"
	appcache "micro-one-api/platform/cache"
	"micro-one-api/platform/database/xdb"
	"micro-one-api/platform/events"
	appgrpc "micro-one-api/platform/grpc"
	applogger "micro-one-api/platform/logging"
	"micro-one-api/platform/metrics"
	appmiddleware "micro-one-api/platform/middleware"
	appregistry "micro-one-api/platform/registry"
	appauth "micro-one-api/platform/security/auth"
	apptls "micro-one-api/platform/tls"

	subscriptionbiz "micro-one-api/domain/subscription/biz"
	subscriptiondata "micro-one-api/domain/subscription/data"
	relaycredential "micro-one-api/domain/upstream/credential"
	relayadaptor "micro-one-api/internal/adaptor"

	"github.com/go-kratos/kratos/v2/config"
	xconfig "micro-one-api/platform/config"
)

// ProviderSet declares the relay-gateway providers. loadConfig lives in
// config_loader.go and the helper functions (newModelMapper, newRetryPolicy,
// createAuthenticatedClient, etc.) live in relay_helpers.go so they are
// visible under both build tags.
//
// relay-gateway's wiring is more complex than the other services (conditional
// client construction, environment-variable-driven configuration, etc.), so
// newApp constructs the provider factory, relay usecase, and HTTP server
// internally rather than declaring them as separate Wire providers.
var ProviderSet = wire.NewSet(
	loadConfig,
	newApp,
)

func InitApp(confPath string) (*kratos.App, func(), error) {
	panic(wire.Build(
		ProviderSet,
	))
}

func newApp(cfg *relaycfg.Config) (*kratos.App, func(), error) {
	tlsConfig := apptls.LoadTLSConfig()
	enableAuth := os.Getenv("ENABLE_AUTH") != "false"
	var serviceAuth *appauth.ServiceAuthConfig
	if enableAuth {
		var err error
		serviceAuth, err = appauth.LoadServiceAuthConfig()
		if err != nil {
			return nil, nil, fmt.Errorf("load service auth config: %w", err)
		}
	}

	providerTimeout := apptimeout.GetUpstreamTimeout()
	if timeoutStr := os.Getenv("RELAY_PROVIDER_TIMEOUT"); timeoutStr != "" {
		if duration, err := time.ParseDuration(timeoutStr); err == nil {
			providerTimeout = duration
		}
	}

	discovery, err := appregistry.NewDiscovery(cfg.Registry)
	if err != nil {
		return nil, nil, fmt.Errorf("create discovery: %w", err)
	}
	registrar, err := appregistry.NewRegistrar(cfg.Registry)
	if err != nil {
		return nil, nil, fmt.Errorf("create registrar: %w", err)
	}

	resolver := appregistry.NewResolver(discovery)
	resolver.SetStatic("identity-service", cfg.Clients.Identity.Endpoint)
	resolver.SetStatic("channel-service", cfg.Clients.Channel.Endpoint)
	resolver.SetStatic("billing-service", cfg.Clients.Billing.Endpoint)
	resolver.SetStatic("log-service", cfg.Clients.Log.Endpoint)

	var identityConn, channelConn, billingConn, logConn *grpc.ClientConn
	var identityClient identityv1.IdentityServiceClient
	var channelClient channelv1.ChannelServiceClient
	var billingClient billingv1.BillingServiceClient
	var logClient logv1.LogServiceClient

	identityEndpoint, err := resolver.ResolveGRPC(context.Background(), "identity-service")
	if err != nil {
		return nil, nil, fmt.Errorf("resolve identity-service endpoint: %w", err)
	}
	channelEndpoint, err := resolver.ResolveGRPC(context.Background(), "channel-service")
	if err != nil {
		return nil, nil, fmt.Errorf("resolve channel-service endpoint: %w", err)
	}
	billingEndpoint, err := resolver.ResolveGRPC(context.Background(), "billing-service")
	if err != nil {
		return nil, nil, fmt.Errorf("resolve billing-service endpoint: %w", err)
	}
	logEndpoint, err := resolver.ResolveGRPC(context.Background(), "log-service")
	if err != nil {
		return nil, nil, fmt.Errorf("resolve log-service endpoint: %w", err)
	}

	if enableAuth && tlsConfig.Enabled {
		identityConn, err = createAuthenticatedClient(identityEndpoint, tlsConfig, serviceAuth)
		if err != nil {
			return nil, nil, fmt.Errorf("create identity gRPC client: %w", err)
		}
		channelConn, err = createAuthenticatedClient(channelEndpoint, tlsConfig, serviceAuth)
		if err != nil {
			return nil, nil, fmt.Errorf("create channel gRPC client: %w", err)
		}
		billingConn, err = createAuthenticatedClient(billingEndpoint, tlsConfig, serviceAuth)
		if err != nil {
			return nil, nil, fmt.Errorf("create billing gRPC client: %w", err)
		}
		logConn, err = createAuthenticatedClient(logEndpoint, tlsConfig, serviceAuth)
		if err != nil {
			return nil, nil, fmt.Errorf("create log gRPC client: %w", err)
		}
	} else {
		identityConn, err = grpc.NewClient(identityEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return nil, nil, fmt.Errorf("create identity gRPC client: %w", err)
		}
		channelConn, err = grpc.NewClient(channelEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return nil, nil, fmt.Errorf("create channel gRPC client: %w", err)
		}
		billingConn, err = grpc.NewClient(billingEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return nil, nil, fmt.Errorf("create billing gRPC client: %w", err)
		}
		logConn, err = grpc.NewClient(logEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return nil, nil, fmt.Errorf("create log gRPC client: %w", err)
		}
	}

	identityClient = identityv1.NewIdentityServiceClient(identityConn)
	channelClient = channelv1.NewChannelServiceClient(channelConn)
	billingClient = billingv1.NewBillingServiceClient(billingConn)
	logClient = logv1.NewLogServiceClient(logConn)

	resilienceTimeout := parseDurationOrDefault(cfg.Resilience.Timeout, 3*time.Second)
	if cfg.Resilience.Enabled {
		identityClient = relaydata.NewResilientIdentityClient(identityClient, resilienceTimeout)
		channelClient = relaydata.NewResilientChannelClient(channelClient, resilienceTimeout)
		billingClient = relaydata.NewResilientBillingClient(billingClient, resilienceTimeout)
		logClient = relaydata.NewResilientLogClient(logClient, resilienceTimeout)
	}

	providerFactory := relayprovider.NewProviderFactory(providerTimeout)
	relayadaptor.SetProviderFactory(providerFactory)

	identityTTL := parseDurationOrDefault(cfg.HybridAdaptor.GetIdentityTTL(), 24*time.Hour)
	identityService := relayidentity.NewIdentityService(identityTTL)
	relayadaptor.SetIdentityService(identityService)

	accountLookup := relaydata.NewChannelSubscriptionAccountStore(channelClient)
	claudeTokenProvider := relaycredential.NewClaudeTokenProvider(accountLookup)
	codexTokenProvider := relaycredential.NewOpenAITokenProvider(accountLookup)

	tokenFactory := func(platform relayidentity.Platform) relaycredential.TokenProvider {
		switch platform {
		case relayidentity.PlatformClaude:
			return claudeTokenProvider
		case relayidentity.PlatformCodex:
			return codexTokenProvider
		default:
			return nil
		}
	}
	relayadaptor.SetTokenProviderFactory(tokenFactory)

	accountResolver := accountLookup
	oauthHTTPClient := &http.Client{Timeout: providerTimeout}

	var refreshTask *relaycredential.RefreshTask
	if cfg.HybridAdaptor.GetTokenRefreshEnabled() {
		refreshTask = relaycredential.NewRefreshTask(
			map[relaycredential.Platform]relaycredential.TokenProvider{
				relaycredential.PlatformClaude: claudeTokenProvider,
				relaycredential.PlatformCodex:  codexTokenProvider,
			},
			accountLookup,
			func(accountID int64) relaycredential.Platform {
				return accountLookup.PlatformOf(context.Background(), accountID)
			},
			relaycredential.RefreshTaskConfig{
				Interval:                  parseDurationOrDefault(cfg.HybridAdaptor.GetRefreshInterval(), 10*time.Minute),
				Lookahead:                 parseDurationOrDefault(cfg.HybridAdaptor.GetRefreshLookahead(), 24*time.Hour),
				MaxRetries:                cfg.HybridAdaptor.TokenRefresh.MaxRetries,
				RetryBackoff:              time.Duration(cfg.HybridAdaptor.TokenRefresh.RetryBackoffSeconds) * time.Second,
				TempUnschedulableDuration: parseDurationOrDefault(cfg.HybridAdaptor.TokenRefresh.TempUnschedDuration, 10*time.Minute),
				Hook:                      accountLookup,
			},
		)
		refreshTask.Start()
	}

	redisAddr := cfg.Redis.Addr
	redisPassword := cfg.Redis.Password
	if redisAddr == "" {
		redisAddr = cfg.OpenAIWS.RedisAddr
		redisPassword = cfg.OpenAIWS.RedisPassword
	}
	redisClient := xdb.NewRedisClient(redisAddr, redisPassword)
	eventBus := events.NewConfiguredEventBus(redisClient, "relay-gateway")
	authLoader := appcache.NewAuthCacheLoader(identityClient, nil, resilienceTimeout)
	authCache, err := appcache.NewAuthCache(redisClient, nil, authLoader.Load)
	if err != nil {
		return nil, nil, fmt.Errorf("create auth cache: %w", err)
	}
	identityClient = relaydata.NewCachedIdentityClient(identityClient, authCache)

	if cfg.ChannelCache.GetChannelCacheEnabled() {
		channelLoader := appcache.NewChannelCacheLoader(channelClient, nil, resilienceTimeout)
		channelCache, err := appcache.NewChannelCache(redisClient, nil, channelLoader.Load)
		if err != nil {
			return nil, nil, fmt.Errorf("create channel cache: %w", err)
		}
		channelClient = relaydata.NewCachedChannelClient(channelClient, channelCache)
	}

	modelMapper := newModelMapper(cfg)
	var modelReloadStop func()
	if mm := modelMapper; mm != nil {
		// Phase 2.5 — models.yaml hot reload. The callback re-reads the file
		// and atomically swaps the mapper's snapshot; a parse/validation
		// failure is logged and the previous snapshot remains in effect.
		if mp := modelsConfigPath(cfg); mp != "" {
			modelReloadStop, _ = xconfig.SubscribeFile(mp, func(_ *config.KeyValue) {
				if err := mm.Reload(); err != nil {
					applogger.Log.Warn("models.yaml hot reload failed; keeping previous snapshot", zap.String("path", mp), zap.Error(err))
					return
				}
				applogger.Log.Info("models.yaml hot-reloaded", zap.String("path", mp))
			})
		}
	}
	retryPolicy := newRetryPolicy(cfg)

	identityAdapter := relaydata.NewIdentityAdapter(identityClient)
	channelAdapter := relaydata.NewChannelAdapter(channelClient)
	relayUsecase := relaybiz.NewRelayUsecase(identityAdapter, channelAdapter, modelMapper, retryPolicy)
	relayUsecase.SetRuntimeBlocker(relaybiz.NewMemoryRuntimeBlocker())

	httpServer := server.NewHTTPServer(identityClient, channelClient, billingClient, providerFactory, relayUsecase, logClient)
	httpServer.SetHybridAdaptorEnabled(cfg.HybridAdaptor.GetHybridAdaptorEnabled())
	httpServer.SetSubscriptionSessionStickyEnabled(cfg.SessionSticky.GetSessionStickyEnabled())
	httpServer.SetRelayOrchestratorEnabled(cfg.RelayOrchestrator.GetRelayOrchestratorEnabled())
	httpServer.SetSubscriptionAccountResolver(accountResolver)
	httpServer.SetOAuthHTTPClient(oauthHTTPClient)
	httpServer.SetSubscriptionAccountQuotaRecorder(accountLookup)
	httpServer.SetUserRPMLimit(cfg.Subscription.GetUserRPMLimit())
	httpServer.SetRuntimeBlockDurations(
		parseDurationOrDefault(cfg.HybridAdaptor.RuntimeBlock.GetRateLimitedDuration(), 5*time.Second),
		parseDurationOrDefault(cfg.HybridAdaptor.RuntimeBlock.GetUnauthorizedDuration(), 2*time.Minute),
		parseDurationOrDefault(cfg.HybridAdaptor.RuntimeBlock.GetServerErrorDuration(), 2*time.Minute),
		parseDurationOrDefault(cfg.HybridAdaptor.RuntimeBlock.GetOverloadedDuration(), 30*time.Second),
	)
	stopBlockerReporter := func() {}
	if redisClient != nil {
		redisBlocker := relaybiz.NewRedisRuntimeBlocker(redisClient)
		httpServer.SetRuntimeBlocker(redisBlocker)
		stopBlockerReporter = redisBlocker.StartActiveGaugeReporter(
			parseDurationOrDefault(cfg.HybridAdaptor.RuntimeBlock.GetActiveGaugeInterval(), 30*time.Second),
			func(v float64) { metrics.RelayRuntimeBlockActive.Set(v) },
		)
		if redisLimiter := relaybiz.NewRedisAccountConcurrencyLimiter(redisClient); redisLimiter != nil {
			httpServer.SetAccountConcurrencyLimiter(redisLimiter)
		}
		if redisRPMLimiter := relaybiz.NewRedisAccountRPMLimiter(redisClient); redisRPMLimiter != nil {
			httpServer.SetAccountRPMLimiter(redisRPMLimiter)
		}
		if redisUserRPMLimiter := relaybiz.NewRedisUserRPMLimiter(redisClient); redisUserRPMLimiter != nil {
			httpServer.SetUserRPMLimiter(redisUserRPMLimiter)
		}
	}

	var routeMiddleware []func(http.Handler) http.Handler
	if cfg.Subscription.GetSubscriptionEnabled() {
		subscriptionRepo, subErr := subscriptiondata.NewRepositoryFromEnv(os.Getenv("SQL_DRIVER"))
		if subErr != nil {
			return nil, nil, fmt.Errorf("create subscription repository: %w", subErr)
		}
		subscriptionUc := subscriptionbiz.NewSubscriptionUsecase(subscriptionRepo, subscriptionRepo)
		httpServer.SetSubscriptionUsecase(subscriptionUc)
		routeMiddleware = append(routeMiddleware, httpServer.SubscriptionQuotaMiddleware)
	}
	if cfg.Idempotency.Enabled {
		ttl := parseDurationOrDefault(cfg.Idempotency.TTL, 24*time.Hour)
		routeMiddleware = append(routeMiddleware, appmiddleware.NewIdempotencyMiddleware(redisClient, &appmiddleware.IdempotencyConfig{
			Header:    "Idempotency-Key",
			TTL:       ttl,
			CacheKeys: true,
		}).Handler)
	}
	if cfg.Audit.Enabled {
		routeMiddleware = append(routeMiddleware, appaudit.NewMiddleware(appaudit.NewAuditor(true)).Handler)
	}
	httpServer.UseRouteMiddleware(routeMiddleware...)

	{
		// WebSocket timeouts are optional: an unparseable value defaults to
		// the zero duration, which the HTTP server treats as "use builtin
		// default". This is an explicit, safe degradation.
		wsWrite, _ := time.ParseDuration(cfg.OpenAIWS.GetOpenAIWSWriteTimeout())
		wsIdle, _ := time.ParseDuration(cfg.OpenAIWS.GetOpenAIWSIdleTimeout())
		wsDial, _ := time.ParseDuration(cfg.OpenAIWS.GetOpenAIWSDialTimeout())
		wsFirst, _ := time.ParseDuration(cfg.OpenAIWS.GetOpenAIWSFirstMessageTimeout())
		// Phase 3.3: graceful-drain config must be set before the connection
		// pool is built so the tracker is created with the configured
		// DrainTimeout / CloseTimeout. Empty / unparseable values fall back to
		// the platform defaults (DrainTimeout=30s, CloseTimeout=10s).
		wsDrain, _ := time.ParseDuration(cfg.OpenAIWS.GetOpenAIWSDrainTimeout())
		if wsDrain > 0 {
			httpServer.SetOpenAIWSDrainConfig(appwsDrainConfig(wsDrain))
		}
		httpServer.SetOpenAIWSTimeouts(wsWrite, wsIdle, wsDial, wsFirst)
		httpServer.SetOpenAIWSConnPool()
		httpServer.SetOpenAIWSPoolConfig(
			cfg.OpenAIWS.GetOpenAIWSMaxConnsPerChannel(),
			cfg.OpenAIWS.GetOpenAIWSFailoverMaxSwitches(),
			parseDurationOrDefault(cfg.OpenAIWS.GetOpenAIWSStickyTTL(), time.Hour),
		)
		httpServer.SetOpenAIWSStickyStore(redisClient)
	}

	srv := newKratosHTTPServer(cfg, httpServer, providerTimeout)

	grpcSvc := relayservice.NewRelayGrpcService(identityClient, channelClient, billingClient, providerFactory, relayUsecase)
	var relayGRPCOpts []grpc.ServerOption
	if cfg.MTLS.Enabled {
		mtlsOpts, mtlsErr := appgrpc.MTLSServerOptions(cfg.MTLS.CertFile, cfg.MTLS.KeyFile, cfg.MTLS.CAFile)
		if mtlsErr != nil {
			return nil, nil, fmt.Errorf("create relay mTLS server options: %w", mtlsErr)
		}
		relayGRPCOpts = append(relayGRPCOpts, mtlsOpts...)
	}
	grpcSrv := server.NewGRPCServer(cfg.Server.GRPC.Addr, grpcSvc, relayGRPCOpts...)

	// Phase 3.3: graceful drain. On SIGTERM the BeforeStop hook drains the
	// tracked WebSocket connections for up to drain_timeout, then the kratos
	// StopTimeout (slightly larger than the drain window) bounds the whole
	// shutdown so an unresponsive client cannot stall termination indefinitely.
	drainTimeout := parseDurationOrDefault(cfg.OpenAIWS.GetOpenAIWSDrainTimeout(), 30*time.Second)
	stopTimeout := drainTimeout + 10*time.Second

	kratosOpts := []kratos.Option{
		kratos.Name("relay-gateway"),
		kratos.Server(srv, grpcSrv),
		kratos.StopTimeout(stopTimeout),
		kratos.BeforeStop(func(ctx context.Context) error {
			drainCtx, cancel := context.WithTimeout(ctx, drainTimeout)
			defer cancel()
			_ = httpServer.DrainWSConnections(drainCtx)
			return nil
		}),
	}
	if registrar != nil {
		kratosOpts = append(kratosOpts, kratos.Registrar(registrar))
	}
	app := kratos.New(kratosOpts...)

	applogger.Log.Info("relay-gateway starting", zap.String("http_addr", cfg.Server.HTTP.Addr))

	cleanup := func() {
		if modelReloadStop != nil {
			modelReloadStop()
		}
		if refreshTask != nil {
			refreshTask.Stop()
		}
		stopBlockerReporter()
		if authCache != nil {
			_ = authCache.Close()
		}
		if closer, ok := eventBus.(interface{ Close() error }); ok {
			_ = closer.Close()
		}
		if redisClient != nil {
			_ = redisClient.Close()
		}
		identityConn.Close()
		channelConn.Close()
		billingConn.Close()
		logConn.Close()
	}

	return app, cleanup, nil
}
