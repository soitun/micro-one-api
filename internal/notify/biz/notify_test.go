package biz

import (
	"context"
	"testing"
	"time"
)

type mockNotifyRepo struct {
	entries map[int64]*Notification
	seq     int64
}

func (m *mockNotifyRepo) Create(ctx context.Context, n *Notification) error {
	m.seq++
	n.ID = m.seq
	m.entries[n.ID] = n
	return nil
}

func (m *mockNotifyRepo) Get(ctx context.Context, id int64) (*Notification, error) {
	n, ok := m.entries[id]
	if !ok {
		return nil, ErrNotificationNotFound
	}
	return n, nil
}

func (m *mockNotifyRepo) List(ctx context.Context, page, pageSize int32, notifyType, status string) ([]*Notification, int64, error) {
	var result []*Notification
	for _, n := range m.entries {
		if notifyType != "" && n.Type != notifyType {
			continue
		}
		if status != "" && n.Status != status {
			continue
		}
		result = append(result, n)
	}
	total := int64(len(result))
	start := int((page - 1) * pageSize)
	if start >= len(result) {
		return nil, total, nil
	}
	end := start + int(pageSize)
	if end > len(result) {
		end = len(result)
	}
	return result[start:end], total, nil
}

func (m *mockNotifyRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	n, ok := m.entries[id]
	if !ok {
		return ErrNotificationNotFound
	}
	n.Status = status
	if status == NotifyStatusSent {
		n.SentAt = time.Now()
	}
	return nil
}

func newMockNotifyRepo() *mockNotifyRepo {
	return &mockNotifyRepo{entries: make(map[int64]*Notification)}
}

func TestNotifyUsecase_CreateNotification(t *testing.T) {
	repo := newMockNotifyRepo()
	uc := NewNotifyUsecase(repo)

	t.Run("success", func(t *testing.T) {
		n, err := uc.CreateNotification(context.Background(), NotifyTypeWebhook, "https://example.com", "alert", "server down")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if n.ID == 0 {
			t.Fatal("expected ID to be set")
		}
		if n.Status != NotifyStatusPending {
			t.Fatalf("expected pending, got %s", n.Status)
		}
		if n.CreatedAt.IsZero() {
			t.Fatal("expected CreatedAt to be set")
		}
		if n.Type != NotifyTypeWebhook {
			t.Fatalf("expected webhook, got %s", n.Type)
		}
		if n.Recipient != "https://example.com" {
			t.Fatalf("unexpected recipient: %s", n.Recipient)
		}
	})

	t.Run("empty type", func(t *testing.T) {
		_, err := uc.CreateNotification(context.Background(), "", "r", "s", "c")
		if err != ErrInvalidNotification {
			t.Fatalf("expected ErrInvalidNotification, got %v", err)
		}
	})

	t.Run("empty recipient", func(t *testing.T) {
		_, err := uc.CreateNotification(context.Background(), NotifyTypeEmail, "", "s", "c")
		if err != ErrInvalidNotification {
			t.Fatalf("expected ErrInvalidNotification, got %v", err)
		}
	})
}

func TestNotifyUsecase_GetNotification(t *testing.T) {
	repo := newMockNotifyRepo()
	uc := NewNotifyUsecase(repo)

	n, _ := uc.CreateNotification(context.Background(), NotifyTypeEmail, "a@b.com", "hi", "hello")

	t.Run("success", func(t *testing.T) {
		got, err := uc.GetNotification(context.Background(), n.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Subject != "hi" {
			t.Fatalf("expected hi, got %s", got.Subject)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := uc.GetNotification(context.Background(), 999)
		if err != ErrNotificationNotFound {
			t.Fatalf("expected ErrNotificationNotFound, got %v", err)
		}
	})
}

func TestNotifyUsecase_ListNotifications(t *testing.T) {
	repo := newMockNotifyRepo()
	uc := NewNotifyUsecase(repo)

	uc.CreateNotification(context.Background(), NotifyTypeWebhook, "a", "s1", "c1")
	uc.CreateNotification(context.Background(), NotifyTypeEmail, "b", "s2", "c2")
	uc.CreateNotification(context.Background(), NotifyTypeWebhook, "c", "s3", "c3")

	t.Run("all", func(t *testing.T) {
		items, total, err := uc.ListNotifications(context.Background(), 1, 50, "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 3 {
			t.Fatalf("expected 3, got %d", total)
		}
		if len(items) != 3 {
			t.Fatalf("expected 3, got %d", len(items))
		}
	})

	t.Run("filter by type", func(t *testing.T) {
		items, total, err := uc.ListNotifications(context.Background(), 1, 50, NotifyTypeWebhook, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 2 {
			t.Fatalf("expected 2, got %d", total)
		}
		if len(items) != 2 {
			t.Fatalf("expected 2, got %d", len(items))
		}
	})

	t.Run("filter by status", func(t *testing.T) {
		items, total, err := uc.ListNotifications(context.Background(), 1, 50, "", NotifyStatusPending)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 3 {
			t.Fatalf("expected 3, got %d", total)
		}
		if len(items) != 3 {
			t.Fatalf("expected 3, got %d", len(items))
		}
	})

	t.Run("normalizes page", func(t *testing.T) {
		_, total, err := uc.ListNotifications(context.Background(), 0, 50, "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 3 {
			t.Fatalf("expected 3, got %d", total)
		}
	})

	t.Run("normalizes pageSize", func(t *testing.T) {
		_, _, err := uc.ListNotifications(context.Background(), 1, 0, "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestNotifyUsecase_MarkSent(t *testing.T) {
	repo := newMockNotifyRepo()
	uc := NewNotifyUsecase(repo)

	n, _ := uc.CreateNotification(context.Background(), NotifyTypeEvent, "svc", "s", "c")

	err := uc.MarkSent(context.Background(), n.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := uc.GetNotification(context.Background(), n.ID)
	if got.Status != NotifyStatusSent {
		t.Fatalf("expected sent, got %s", got.Status)
	}
}

func TestNotifyUsecase_MarkFailed(t *testing.T) {
	repo := newMockNotifyRepo()
	uc := NewNotifyUsecase(repo)

	n, _ := uc.CreateNotification(context.Background(), NotifyTypeEvent, "svc", "s", "c")

	err := uc.MarkFailed(context.Background(), n.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := uc.GetNotification(context.Background(), n.ID)
	if got.Status != NotifyStatusFailed {
		t.Fatalf("expected failed, got %s", got.Status)
	}
}

func TestNotifyConstants(t *testing.T) {
	if NotifyTypeWebhook != "webhook" {
		t.Fatalf("expected webhook, got %s", NotifyTypeWebhook)
	}
	if NotifyTypeEmail != "email" {
		t.Fatalf("expected email, got %s", NotifyTypeEmail)
	}
	if NotifyTypeEvent != "event" {
		t.Fatalf("expected event, got %s", NotifyTypeEvent)
	}
	if NotifyStatusPending != "pending" {
		t.Fatalf("expected pending, got %s", NotifyStatusPending)
	}
	if NotifyStatusSent != "sent" {
		t.Fatalf("expected sent, got %s", NotifyStatusSent)
	}
	if NotifyStatusFailed != "failed" {
		t.Fatalf("expected failed, got %s", NotifyStatusFailed)
	}
}
