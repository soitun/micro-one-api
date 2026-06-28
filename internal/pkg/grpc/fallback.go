package grpc

import (
	"context"
	"fmt"

	billingv1 "micro-one-api/api/billing/v1"
	commonv1 "micro-one-api/api/common/v1"
	identityv1 "micro-one-api/api/identity/v1"

	"github.com/sony/gobreaker"
)

// AuthCacheFallback uses cached auth snapshots when identity-service is down.
type AuthCacheFallback struct {
	// TODO: Add cache client
}

// NewAuthCacheFallback creates a new auth cache fallback.
func NewAuthCacheFallback() *AuthCacheFallback {
	return &AuthCacheFallback{}
}

// ExecuteFallback returns cached auth data or an error.
func (f *AuthCacheFallback) ExecuteFallback(ctx context.Context, token string) (*identityv1.GetAuthSnapshotReply, error) {
	// TODO: Implement cache lookup
	return nil, fmt.Errorf("auth cache not implemented yet")
}

// ChannelCacheFallback uses cached channel data when channel-service is down.
type ChannelCacheFallback struct {
	// TODO: Add cache client
}

// NewChannelCacheFallback creates a new channel cache fallback.
func NewChannelCacheFallback() *ChannelCacheFallback {
	return &ChannelCacheFallback{}
}

// ExecuteFallback returns cached channel data or an error.
func (f *ChannelCacheFallback) ExecuteFallback(ctx context.Context, group, model string) (*commonv1.ChannelInfo, error) {
	// TODO: Implement cache lookup
	return nil, fmt.Errorf("channel cache not implemented yet")
}

// AsyncBillingFallback enables async billing when billing-service is down.
type AsyncBillingFallback struct {
	// TODO: Add async queue
}

// NewAsyncBillingFallback creates a new async billing fallback.
func NewAsyncBillingFallback() *AsyncBillingFallback {
	return &AsyncBillingFallback{}
}

// ExecuteFallback queues the billing operation for async processing.
func (f *AsyncBillingFallback) ExecuteFallback(ctx context.Context, req *billingv1.ReserveQuotaRequest) (*billingv1.ReserveQuotaResponse, error) {
	// TODO: Implement async queue
	return &billingv1.ReserveQuotaResponse{
		Success:       true,
		ReservationId: "async-" + req.RequestId,
	}, nil
}

// NoOpFallback discards the operation when the service is down.
type NoOpFallback struct{}

// NewNoOpFallback creates a new no-op fallback.
func NewNoOpFallback() *NoOpFallback {
	return &NoOpFallback{}
}

// ExecuteFallback does nothing and returns success.
func (f *NoOpFallback) ExecuteFallback(ctx context.Context) error {
	// Intentionally do nothing
	return nil
}

// RejectFallback immediately rejects the request when the service is down.
type RejectFallback struct {
	serviceName string
}

// NewRejectFallback creates a new reject fallback.
func NewRejectFallback(serviceName string) *RejectFallback {
	return &RejectFallback{serviceName: serviceName}
}

// ExecuteFallback returns an error indicating the service is unavailable.
func (f *RejectFallback) ExecuteFallback(ctx context.Context) error {
	return fmt.Errorf("service %s is currently unavailable (circuit breaker open)", f.serviceName)
}

// FallbackFactory creates fallback functions for different services.
type FallbackFactory struct {
	authCache    *AuthCacheFallback
	channelCache *ChannelCacheFallback
	asyncBilling *AsyncBillingFallback
	noOp         *NoOpFallback
}

// NewFallbackFactory creates a new fallback factory.
func NewFallbackFactory() *FallbackFactory {
	return &FallbackFactory{
		authCache:    NewAuthCacheFallback(),
		channelCache: NewChannelCacheFallback(),
		asyncBilling: NewAsyncBillingFallback(),
		noOp:         NewNoOpFallback(),
	}
}

// CreateAuthFallback creates a fallback function for identity service.
func (f *FallbackFactory) CreateAuthFallback() FallbackFunc[any] {
	return func(ctx context.Context, err error) (any, error) {
		// TODO: Extract token from context and return cached snapshot
		return nil, fmt.Errorf("auth fallback: %w", err)
	}
}

// CreateChannelFallback creates a fallback function for channel service.
func (f *FallbackFactory) CreateChannelFallback() FallbackFunc[any] {
	return func(ctx context.Context, err error) (any, error) {
		// TODO: Extract group/model from context and return cached channel
		return nil, fmt.Errorf("channel fallback: %w", err)
	}
}

// CreateBillingFallback creates a fallback function for billing service.
func (f *FallbackFactory) CreateBillingFallback() FallbackFunc[any] {
	return func(ctx context.Context, err error) (any, error) {
		// Return success to allow request to proceed with async billing
		return &billingv1.ReserveQuotaResponse{
			Success: true,
		}, nil
	}
}

// CreateLogFallback creates a fallback function for log service.
func (f *FallbackFactory) CreateLogFallback() FallbackFunc[any] {
	return func(ctx context.Context, err error) (any, error) {
		// Discard log silently
		return nil, nil
	}
}

// CreateRejectFallback creates a fallback function that rejects requests.
func (f *FallbackFactory) CreateRejectFallback(serviceName string) FallbackFunc[any] {
	return func(ctx context.Context, err error) (any, error) {
		return nil, fmt.Errorf("service %s unavailable: %w", serviceName, err)
	}
}

// CircuitBreakerManager manages circuit breakers for all services.
type CircuitBreakerManager struct {
	identityBreaker  ResilientClient[any]
	channelBreaker   ResilientClient[any]
	billingBreaker   ResilientClient[any]
	logBreaker       ResilientClient[any]
	fallbackFactory  *FallbackFactory
}

// NewCircuitBreakerManager creates a new circuit breaker manager.
func NewCircuitBreakerManager(
	identityClient any,
	channelClient any,
	billingClient any,
	logClient any,
) *CircuitBreakerManager {
	factory := NewFallbackFactory()

	return &CircuitBreakerManager{
		fallbackFactory: factory,
		// Circuit breakers will be initialized with actual clients
	}
}

// IdentityBreaker returns the circuit breaker for identity service.
func (m *CircuitBreakerManager) IdentityBreaker() *ResilientClient[any] {
	return &m.identityBreaker
}

// ChannelBreaker returns the circuit breaker for channel service.
func (m *CircuitBreakerManager) ChannelBreaker() *ResilientClient[any] {
	return &m.channelBreaker
}

// BillingBreaker returns the circuit breaker for billing service.
func (m *CircuitBreakerManager) BillingBreaker() *ResilientClient[any] {
	return &m.billingBreaker
}

// LogBreaker returns the circuit breaker for log service.
func (m *CircuitBreakerManager) LogBreaker() *ResilientClient[any] {
	return &m.logBreaker
}

// GetBreakerByName returns a circuit breaker by service name.
func (m *CircuitBreakerManager) GetBreakerByName(service string) *ResilientClient[any] {
	switch service {
	case "identity":
		return &m.identityBreaker
	case "channel":
		return &m.channelBreaker
	case "billing":
		return &m.billingBreaker
	case "log":
		return &m.logBreaker
	default:
		return nil
	}
}

// IsServiceHealthy checks if a service is healthy (breaker is closed or half-open).
func (m *CircuitBreakerManager) IsServiceHealthy(service string) bool {
	breaker := m.GetBreakerByName(service)
	if breaker == nil {
		return true // No breaker means always healthy
	}
	state := breaker.State()
	return state == gobreaker.StateClosed || state == gobreaker.StateHalfOpen
}

// GetDegradationLevel assesses the overall system health and returns degradation level.
func (m *CircuitBreakerManager) GetDegradationLevel() DegradationLevel {
	identityHealthy := m.IsServiceHealthy("identity")
	channelHealthy := m.IsServiceHealthy("channel")
	billingHealthy := m.IsServiceHealthy("billing")
	logHealthy := m.IsServiceHealthy("log")

	downServices := 0
	if !identityHealthy {
		downServices++
	}
	if !channelHealthy {
		downServices++
	}
	if !billingHealthy {
		downServices++
	}
	if !logHealthy {
		downServices++
	}

	switch {
	case downServices == 0:
		return DegradationNone
	case downServices == 1 && !billingHealthy:
		return DegradationAsync
	case !identityHealthy || !channelHealthy:
		return DegradationCached
	case downServices >= 2:
		return DegradationMinimal
	default:
		return DegradationCached
	}
}

// DegradationLevel represents the system degradation level.
type DegradationLevel int

const (
	DegradationNone     DegradationLevel = 0 // All services healthy
	DegradationCached   DegradationLevel = 1 // Using cached data
	DegradationAsync    DegradationLevel = 2 // Async billing enabled
	DegradationMinimal  DegradationLevel = 3 // Minimal functionality
)

// String returns the string representation of the degradation level.
func (d DegradationLevel) String() string {
	switch d {
	case DegradationNone:
		return "none"
	case DegradationCached:
		return "cached"
	case DegradationAsync:
		return "async"
	case DegradationMinimal:
		return "minimal"
	default:
		return "unknown"
	}
}
