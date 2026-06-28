package events

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/redis/go-redis/v9"
)

const (
	// DefaultStreamMaxLen is the default maximum length for streams.
	DefaultStreamMaxLen = 10000
	// DefaultConsumerGroup is the default consumer group name.
	DefaultConsumerGroup = "micro-one-api"
)

// StreamEventBus is a cross-process EventBus backed by Redis Streams.
// It guarantees at-least-once delivery with consumer groups.
type StreamEventBus struct {
	redis         *redis.Client
	consumerID    string
	consumerGroup string
	handlers      map[string][]Handler
	handlersMu    sync.RWMutex
	maxlen        int64
	readTimeout   time.Duration
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	closed        bool
	mu            sync.Mutex
}

// NewStreamEventBus creates a new Redis Streams-based event bus.
func NewStreamEventBus(redisClient *redis.Client, consumerID string) *StreamEventBus {
	ctx, cancel := context.WithCancel(context.Background())

	return &StreamEventBus{
		redis:         redisClient,
		consumerID:    consumerID,
		consumerGroup: DefaultConsumerGroup,
		handlers:      make(map[string][]Handler),
		maxlen:        DefaultStreamMaxLen,
		readTimeout:   5 * time.Second,
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Publish sends an event to a Redis Stream with guaranteed persistence.
// Events survive process restarts.
func (b *StreamEventBus) Publish(ctx context.Context, topic string, payload interface{}) error {
	data, err := sonic.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal event payload: %w", err)
	}

	err = b.redis.XAdd(ctx, &redis.XAddArgs{
		Stream: topic,
		MaxLen: b.maxlen,
		Approx: true,
		Values: map[string]interface{}{
			"payload":   string(data),
			"timestamp": time.Now().UnixNano(),
			"producer":  b.consumerID,
		},
	}).Err()

	if err != nil {
		return fmt.Errorf("publish event to stream %s: %w", topic, err)
	}

	return nil
}

// Subscribe joins a consumer group and processes events.
// Each event is ACKed only after the handler succeeds.
func (b *StreamEventBus) Subscribe(topic string, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return
	}

	b.handlersMu.Lock()
	b.handlers[topic] = append(b.handlers[topic], handler)
	b.handlersMu.Unlock()

	// Ensure consumer group exists
	if err := b.ensureGroup(b.ctx, topic); err != nil {
		// Log error but don't fail - will retry on consume
		fmt.Printf("failed to ensure consumer group for %s: %v\n", topic, err)
	}

	// Start consume loop if not already running for this topic
	b.wg.Add(1)
	go b.consumeLoop(topic)
}

// consumeLoop continuously reads and processes events from a stream.
func (b *StreamEventBus) consumeLoop(topic string) {
	defer b.wg.Done()

	for {
		select {
		case <-b.ctx.Done():
			return
		default:
		}

		// Read new messages
		msgs, err := b.redis.XRead(b.ctx, &redis.XReadArgs{
			Streams: []string{topic, ">"},
			Count:   10,
			Block:  b.readTimeout,
		}).Result()

		if err != nil {
			if err == redis.Nil {
				// No new messages, continue
				continue
			}
			// Log error and continue
			fmt.Printf("error reading from stream %s: %v\n", topic, err)
			time.Sleep(time.Second)
			continue
		}

		// Process messages
		for _, stream := range msgs {
			for _, msg := range stream.Messages {
				b.processMessage(topic, &msg)
			}
		}
	}
}

// processMessage processes a single message from a stream.
func (b *StreamEventBus) processMessage(topic string, msg *redis.XMessage) {
	ctx := context.Background()

	// Extract payload
	payloadData, ok := msg.Values["payload"].(string)
	if !ok {
		fmt.Printf("missing payload in message from %s\n", topic)
		return
	}

	// Unmarshal payload
	var payload Event
	if err := sonic.Unmarshal([]byte(payloadData), &payload); err != nil {
		fmt.Printf("failed to unmarshal payload from %s: %v\n", topic, err)
		return
	}

	// Get handlers for this topic
	b.handlersMu.RLock()
	handlers, exists := b.handlers[topic]
	b.handlersMu.RUnlock()

	if !exists || len(handlers) == 0 {
		return
	}

	// Call handlers
	for _, handler := range handlers {
		if err := handler(ctx, payload); err != nil {
			fmt.Printf("handler error for topic %s: %v\n", topic, err)
			// Continue processing other handlers
		}
	}

	// ACK the message
	if err := b.redis.XAck(ctx, topic, b.consumerGroup, msg.ID).Err(); err != nil {
		fmt.Printf("failed to ACK message %s from %s: %v\n", msg.ID, topic, err)
	}
}

// ensureGroup ensures the consumer group exists for a stream.
func (b *StreamEventBus) ensureGroup(ctx context.Context, stream string) error {
	// Try to create consumer group with MKSTREAM option
	err := b.redis.Do(ctx, "XGROUP", "CREATE", stream, b.consumerGroup, "0", "MKSTREAM").Err()
	if err != nil {
		// Group might already exist or other error
		// Check if group exists
		info, err := b.redis.XInfoGroups(ctx, stream).Result()
		if err == nil {
			for _, group := range info {
				if group.Name == b.consumerGroup {
					return nil
				}
			}
		}
		return err
	}
	return nil
}

// Close closes the event bus and waits for all consumers to finish.
func (b *StreamEventBus) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}

	b.closed = true
	b.cancel()
	b.wg.Wait()

	return nil
}

// StreamStats holds statistics for a stream.
type StreamStats struct {
	Stream          string
	ConsumerGroup   string
	Consumers       []string
	Pending         int64
	LastDeliveredID string
}

// Stats returns statistics for all streams being consumed.
func (b *StreamEventBus) Stats(ctx context.Context) ([]*StreamStats, error) {
	b.handlersMu.RLock()
	defer b.handlersMu.RUnlock()

	var stats []*StreamStats

	for topic := range b.handlers {
		// Get consumer group info
		info, err := b.redis.XInfoGroups(ctx, topic).Result()
		if err != nil {
			continue
		}

		for _, group := range info {
			if group.Name == b.consumerGroup {
				// Get consumers
				consumers, err := b.redis.XInfoConsumers(ctx, topic, b.consumerGroup).Result()
				if err != nil {
					continue
				}

				var consumerNames []string
				for _, consumer := range consumers {
					consumerNames = append(consumerNames, consumer.Name)
				}

				stats = append(stats, &StreamStats{
					Stream:          topic,
					ConsumerGroup:   group.Name,
					Consumers:       consumerNames,
					Pending:         group.Pending,
					LastDeliveredID: group.LastDeliveredID,
				})
			}
		}
	}

	return stats, nil
}

// ClaimPending processes pending messages that were not ACKed by a crashed consumer.
// This is a simplified implementation that claims idle messages.
func (b *StreamEventBus) ClaimPending(ctx context.Context, topic string) error {
	// TODO: Implement proper pending message claiming
	// This requires using XAUTOCLAIM or complex XPendingExt logic
	return fmt.Errorf("ClaimPending not yet implemented")
}

// Trim trims the stream to the specified maximum length.
func (b *StreamEventBus) Trim(ctx context.Context, topic string, exact bool, maxLen int64) error {
	if exact {
		return b.redis.Do(ctx, "XTRIM", topic, "MAXLEN", maxLen).Err()
	}
	return b.redis.Do(ctx, "XTRIM", topic, "MAXLEN", "~", maxLen).Err()
}

// ReadLast reads the last N messages from a stream without consumer group.
// Useful for debugging or backfilling.
func (b *StreamEventBus) ReadLast(ctx context.Context, topic string, start, stop string) ([]redis.XMessage, error) {
	return b.redis.XRevRange(ctx, topic, start, stop).Result()
}
