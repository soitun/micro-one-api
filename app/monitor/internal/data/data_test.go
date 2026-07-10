package data

import (
	"context"
	"testing"
	"time"

	"micro-one-api/app/monitor/internal/biz"
)

func TestMemoryRepository_SaveAndGetHealthCheck(t *testing.T) {
	repo := newMemoryRepository()

	check := &biz.HealthCheck{
		ServiceName:  "billing-service",
		Status:       biz.HealthStatusHealthy,
		ResponseTime: 25,
		CheckedAt:    time.Now(),
	}

	err := repo.SaveHealthCheck(context.Background(), check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if check.ID == 0 {
		t.Fatal("expected ID to be assigned")
	}

	latest, err := repo.GetLatestHealthCheck(context.Background(), "billing-service")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if latest.ServiceName != "billing-service" {
		t.Fatalf("expected billing-service, got %s", latest.ServiceName)
	}
}

func TestMemoryRepository_GetLatestHealthCheck_NotFound(t *testing.T) {
	repo := newMemoryRepository()
	_, err := repo.GetLatestHealthCheck(context.Background(), "nonexistent")
	if err != biz.ErrHealthCheckNotFound {
		t.Fatalf("expected ErrHealthCheckNotFound, got %v", err)
	}
}

func TestMemoryRepository_ListHealthChecks(t *testing.T) {
	repo := newMemoryRepository()
	_ = repo.SaveHealthCheck(context.Background(), &biz.HealthCheck{ServiceName: "s1", Status: biz.HealthStatusHealthy, CheckedAt: time.Now()})
	_ = repo.SaveHealthCheck(context.Background(), &biz.HealthCheck{ServiceName: "s2", Status: biz.HealthStatusUnhealthy, CheckedAt: time.Now()})

	t.Run("all", func(t *testing.T) {
		checks, total, err := repo.ListHealthChecks(context.Background(), "", 1, 20)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// 1 from default + 2 created
		if total != 3 {
			t.Fatalf("expected 3, got %d", total)
		}
		if len(checks) != 3 {
			t.Fatalf("expected 3, got %d", len(checks))
		}
	})

	t.Run("filter by service", func(t *testing.T) {
		checks, total, err := repo.ListHealthChecks(context.Background(), "s1", 1, 20)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 1 {
			t.Fatalf("expected 1, got %d", total)
		}
		if len(checks) != 1 || checks[0].ServiceName != "s1" {
			t.Fatalf("unexpected checks: %+v", checks)
		}
	})
}

func TestMemoryRepository_AlertRuleCRUD(t *testing.T) {
	repo := newMemoryRepository()

	rule := &biz.AlertRule{
		Name:        "high-latency",
		ServiceName: "relay-gateway",
		Metric:      "response_time",
		Threshold:   500,
		Operator:    "gt",
		Duration:    60,
		Enabled:     true,
		CreatedAt:   time.Now(),
	}

	t.Run("create", func(t *testing.T) {
		err := repo.CreateAlertRule(context.Background(), rule)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rule.ID == 0 {
			t.Fatal("expected ID to be assigned")
		}
	})

	t.Run("get", func(t *testing.T) {
		got, err := repo.GetAlertRule(context.Background(), rule.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Name != "high-latency" {
			t.Fatalf("expected high-latency, got %s", got.Name)
		}
	})

	t.Run("update", func(t *testing.T) {
		rule.Threshold = 1000
		err := repo.UpdateAlertRule(context.Background(), rule)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, _ := repo.GetAlertRule(context.Background(), rule.ID)
		if got.Threshold != 1000 {
			t.Fatalf("expected 1000, got %f", got.Threshold)
		}
	})

	t.Run("list", func(t *testing.T) {
		rules, total, err := repo.ListAlertRules(context.Background(), 1, 20)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 1 {
			t.Fatalf("expected 1, got %d", total)
		}
		if len(rules) != 1 {
			t.Fatalf("expected 1, got %d", len(rules))
		}
	})

	t.Run("delete", func(t *testing.T) {
		err := repo.DeleteAlertRule(context.Background(), rule.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		_, err = repo.GetAlertRule(context.Background(), rule.ID)
		if err != biz.ErrAlertRuleNotFound {
			t.Fatalf("expected ErrAlertRuleNotFound, got %v", err)
		}
	})

	t.Run("delete not found", func(t *testing.T) {
		err := repo.DeleteAlertRule(context.Background(), 999)
		if err != biz.ErrAlertRuleNotFound {
			t.Fatalf("expected ErrAlertRuleNotFound, got %v", err)
		}
	})
}
