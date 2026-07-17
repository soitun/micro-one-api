package server

import (
	"time"
)

type openAIWSTimeouts struct {
	writeTimeout        time.Duration
	idleTimeout         time.Duration
	dialTimeout         time.Duration
	firstMessageTimeout time.Duration
}

type openAIWSPoolConfig struct {
	maxConnsPerChannel  int
	failoverMaxSwitches int
	stickyTTL           time.Duration
}

type runtimeBlockConfig struct {
	rateLimited  time.Duration // 429
	unauthorized time.Duration // 401
	serverError  time.Duration // 5xx
	overloaded   time.Duration // 529
}
