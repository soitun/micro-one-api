package biz

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// RuntimeBlock records short-lived account blocks observed while relaying
// upstream traffic. These blocks are operational, not account lifecycle state.
type RuntimeBlock struct {
	AccountID int64
	Until     time.Time
	Reason    string
}

type RuntimeBlockerMetrics struct {
	Blocks     int64
	Hits       int64
	Expired    int64
	Clears     int64
	ActiveSize int64
}

// RuntimeBlocker filters subscription accounts that recently failed at
// runtime. Implementations may be in-memory, Redis-backed, or both.
type RuntimeBlocker interface {
	Block(ctx context.Context, accountID int64, until time.Time, reason string) error
	Clear(ctx context.Context, accountID int64) error
	IsBlocked(ctx context.Context, accountID int64, now time.Time) (RuntimeBlock, bool)
	Metrics() RuntimeBlockerMetrics
}

type NoopRuntimeBlocker struct{}

func (NoopRuntimeBlocker) Block(context.Context, int64, time.Time, string) error { return nil }
func (NoopRuntimeBlocker) Clear(context.Context, int64) error                    { return nil }
func (NoopRuntimeBlocker) IsBlocked(context.Context, int64, time.Time) (RuntimeBlock, bool) {
	return RuntimeBlock{}, false
}
func (NoopRuntimeBlocker) Metrics() RuntimeBlockerMetrics { return RuntimeBlockerMetrics{} }

// MemoryRuntimeBlocker is the default relay-gateway runtime blocker. It keeps
// short TTL blocks local to the process and prunes expired entries on access.
type MemoryRuntimeBlocker struct {
	mu     sync.Mutex
	blocks map[int64]RuntimeBlock

	blockCount  atomic.Int64
	hitCount    atomic.Int64
	expireCount atomic.Int64
	clearCount  atomic.Int64
}

func NewMemoryRuntimeBlocker() *MemoryRuntimeBlocker {
	return &MemoryRuntimeBlocker{blocks: make(map[int64]RuntimeBlock)}
}

func (b *MemoryRuntimeBlocker) Block(_ context.Context, accountID int64, until time.Time, reason string) error {
	if b == nil || accountID <= 0 || until.IsZero() {
		return nil
	}
	b.mu.Lock()
	if b.blocks == nil {
		b.blocks = make(map[int64]RuntimeBlock)
	}
	b.blocks[accountID] = RuntimeBlock{AccountID: accountID, Until: until, Reason: reason}
	b.mu.Unlock()
	b.blockCount.Add(1)
	return nil
}

func (b *MemoryRuntimeBlocker) Clear(_ context.Context, accountID int64) error {
	if b == nil || accountID <= 0 {
		return nil
	}
	b.mu.Lock()
	delete(b.blocks, accountID)
	b.mu.Unlock()
	b.clearCount.Add(1)
	return nil
}

func (b *MemoryRuntimeBlocker) IsBlocked(_ context.Context, accountID int64, now time.Time) (RuntimeBlock, bool) {
	if b == nil || accountID <= 0 {
		return RuntimeBlock{}, false
	}
	if now.IsZero() {
		now = time.Now()
	}
	b.mu.Lock()
	block, ok := b.blocks[accountID]
	if ok && !block.Until.After(now) {
		delete(b.blocks, accountID)
		ok = false
		b.expireCount.Add(1)
	}
	b.mu.Unlock()
	if ok {
		b.hitCount.Add(1)
		return block, true
	}
	return RuntimeBlock{}, false
}

func (b *MemoryRuntimeBlocker) Metrics() RuntimeBlockerMetrics {
	if b == nil {
		return RuntimeBlockerMetrics{}
	}
	b.mu.Lock()
	active := int64(len(b.blocks))
	b.mu.Unlock()
	return RuntimeBlockerMetrics{
		Blocks:     b.blockCount.Load(),
		Hits:       b.hitCount.Load(),
		Expired:    b.expireCount.Load(),
		Clears:     b.clearCount.Load(),
		ActiveSize: active,
	}
}

var _ RuntimeBlocker = (*MemoryRuntimeBlocker)(nil)
var _ RuntimeBlocker = NoopRuntimeBlocker{}
