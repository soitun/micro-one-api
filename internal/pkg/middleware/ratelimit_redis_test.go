package middleware

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRedisRateLimiter_NilClient(t *testing.T) {
	limiter := NewRedisRateLimiter(nil, nil)
	allowed, err := limiter.Allow(context.Background(), "test-key")
	assert.NoError(t, err)
	assert.True(t, allowed)
}

func TestDefaultRedisRateLimitConfig(t *testing.T) {
	cfg := DefaultRedisRateLimitConfig()
	assert.Equal(t, 100, cfg.RequestsPerSecond)
	assert.Equal(t, 200, cfg.Burst)
	assert.Equal(t, "ratelimit:", cfg.KeyPrefix)
}
