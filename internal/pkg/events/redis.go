package events

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisEventBus implements EventBus using Redis Pub/Sub.
// It supports cross-process event delivery and survives service restarts.
type RedisEventBus struct {
	rdb      *redis.Client
	mu       sync.RWMutex
	handlers map[string][]Handler
}

// NewRedisEventBus creates a new Redis-backed EventBus.
func NewRedisEventBus(rdb *redis.Client) *RedisEventBus {
	return &RedisEventBus{
		rdb:      rdb,
		handlers: make(map[string][]Handler),
	}
}

// Publish sends an event to the given topic via Redis Pub/Sub.
func (b *RedisEventBus) Publish(ctx context.Context, topic string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return b.rdb.Publish(ctx, topic, data).Err()
}

// Subscribe registers a handler for the given topic.
// The handler is called locally when messages arrive on the Redis channel.
func (b *RedisEventBus) Subscribe(topic string, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[topic] = append(b.handlers[topic], handler)
}

// StartListening subscribes to all registered topics on Redis and dispatches
// events to local handlers. Call this after all Subscribe() calls.
// Returns a stop function that cancels the background goroutines.
func (b *RedisEventBus) StartListening(ctx context.Context) func() {
	b.mu.RLock()
	topics := make([]string, 0, len(b.handlers))
	for t := range b.handlers {
		topics = append(topics, t)
	}
	b.mu.RUnlock()

	if len(topics) == 0 {
		return func() {}
	}

	ctx, cancel := context.WithCancel(ctx)

	for _, topic := range topics {
		go b.listenTopic(ctx, topic)
	}

	return cancel
}

func (b *RedisEventBus) listenTopic(ctx context.Context, topic string) {
	pubsub := b.rdb.Subscribe(ctx, topic)
	defer pubsub.Close()

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			b.dispatch(ctx, topic, msg.Payload)
		}
	}
}

func (b *RedisEventBus) dispatch(ctx context.Context, topic, payload string) {
	b.mu.RLock()
	handlers := b.handlers[topic]
	b.mu.RUnlock()

	event := Event{
		Topic:     topic,
		Payload:   payload,
		Timestamp: time.Now(),
	}

	for _, h := range handlers {
		_ = h(ctx, event)
	}
}
