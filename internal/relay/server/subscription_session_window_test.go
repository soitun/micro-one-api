package server

import (
	"context"
	"testing"
	"time"
)

func TestSubscriptionSessionWindowStore_RecordExceededAndDedupe(t *testing.T) {
	store := newSubscriptionSessionWindowStore(nil)
	ctx := context.Background()

	if store.Exceeded(ctx, "default", "session-a", 42, 1) {
		t.Fatal("empty window should not be exceeded")
	}
	store.RecordUsage(ctx, "default", "session-a", 42, "reservation-1", 0.75, time.Hour)
	if store.Exceeded(ctx, "default", "session-a", 42, 1) {
		t.Fatal("window below limit should not be exceeded")
	}
	store.RecordUsage(ctx, "default", "session-a", 42, "reservation-1", 0.75, time.Hour)
	if store.Exceeded(ctx, "default", "session-a", 42, 1) {
		t.Fatal("duplicate reservation must not double-count")
	}
	store.RecordUsage(ctx, "default", "session-a", 42, "reservation-2", 0.25, time.Hour)
	if !store.Exceeded(ctx, "default", "session-a", 42, 1) {
		t.Fatal("window at limit should be exceeded")
	}
	if store.Exceeded(ctx, "default", "session-b", 42, 1) {
		t.Fatal("different sessions must have independent windows")
	}
}

func TestSubscriptionSessionWindowStore_Expires(t *testing.T) {
	store := newSubscriptionSessionWindowStore(nil)
	ctx := context.Background()

	store.RecordUsage(ctx, "default", "session-a", 42, "reservation-1", 1, time.Millisecond)
	if !store.Exceeded(ctx, "default", "session-a", 42, 1) {
		t.Fatal("window should initially be exceeded")
	}
	time.Sleep(2 * time.Millisecond)
	if store.Exceeded(ctx, "default", "session-a", 42, 1) {
		t.Fatal("expired window should not be exceeded")
	}
}
