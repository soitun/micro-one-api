package biz

import (
	"context"
	"testing"
	"time"
)

type stubOperationReportRepo struct {
	orderRows []PlanOperationRow
	active    map[int64]int64
	expired   map[int64]int64
	revoked   map[int64]int64
	err       error
}

func (s *stubOperationReportRepo) AggregatePaymentOrdersByPlan(ctx context.Context, startTime, endTime time.Time, planID, groupID int64, userID string) ([]PlanOperationRow, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.orderRows, nil
}
func (s *stubOperationReportRepo) CountSubscriptionsByStatus(ctx context.Context, planID, groupID int64) (map[int64]int64, map[int64]int64, map[int64]int64, error) {
	return s.active, s.expired, s.revoked, s.err
}

func TestSubscriptionReport_AggregatesRowsAndTotals(t *testing.T) {
	repo := &stubOperationReportRepo{
		orderRows: []PlanOperationRow{
			{PlanID: 1, PlanName: "Basic", GroupID: 1, NewPurchaseCount: 5, RenewalCount: 2, RefundCount: 1, RevenueQuota: 700, RefundedQuota: 100},
			{PlanID: 2, PlanName: "Pro", GroupID: 2, NewPurchaseCount: 3, RenewalCount: 1, RefundCount: 0, RevenueQuota: 1200, RefundedQuota: 0},
		},
		active:  map[int64]int64{1: 4, 2: 3},
		expired: map[int64]int64{1: 1},
		revoked: map[int64]int64{1: 1, 2: 0},
	}
	uc := NewSubscriptionReportUsecase(repo)
	uc.now = func() time.Time { return time.Unix(1_700_000_000, 0) }

	report, err := uc.BuildReport(context.Background(), time.Time{}, time.Time{}, 0, 0, "")
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if len(report.Rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(report.Rows))
	}
	if report.Rows[0].ActiveSubscriptions != 4 {
		t.Fatalf("active[0] = %d, want 4", report.Rows[0].ActiveSubscriptions)
	}
	if report.Rows[0].RevokedSubscriptions != 1 {
		t.Fatalf("revoked[0] = %d, want 1", report.Rows[0].RevokedSubscriptions)
	}
	if report.TotalRevenueQuota != 1900 {
		t.Fatalf("total revenue = %d, want 1900", report.TotalRevenueQuota)
	}
	if report.TotalRefundedQuota != 100 {
		t.Fatalf("total refunded = %d, want 100", report.TotalRefundedQuota)
	}
}

func TestSubscriptionReport_DefaultsToLast30Days(t *testing.T) {
	calls := 0
	repo := &stubOperationReportRepo{
		orderRows: []PlanOperationRow{},
		active:    map[int64]int64{},
		expired:   map[int64]int64{},
		revoked:   map[int64]int64{},
	}
	wrapped := &capturingReportRepo{inner: repo, onStart: func(st, et time.Time) {
		calls++
		if et.Sub(st) < 29*24*time.Hour {
			t.Errorf("default window too short: %v", et.Sub(st))
		}
	}}
	uc := NewSubscriptionReportUsecase(wrapped)
	uc.now = func() time.Time { return time.Unix(1_700_000_000, 0) }

	if _, err := uc.BuildReport(context.Background(), time.Time{}, time.Time{}, 0, 0, ""); err != nil {
		t.Fatalf("build: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 aggregation call, got %d", calls)
	}
}

type capturingReportRepo struct {
	inner   *stubOperationReportRepo
	onStart func(start, end time.Time)
}

func (c *capturingReportRepo) AggregatePaymentOrdersByPlan(ctx context.Context, startTime, endTime time.Time, planID, groupID int64, userID string) ([]PlanOperationRow, error) {
	c.onStart(startTime, endTime)
	return c.inner.AggregatePaymentOrdersByPlan(ctx, startTime, endTime, planID, groupID, userID)
}
func (c *capturingReportRepo) CountSubscriptionsByStatus(ctx context.Context, planID, groupID int64) (map[int64]int64, map[int64]int64, map[int64]int64, error) {
	return c.inner.CountSubscriptionsByStatus(ctx, planID, groupID)
}

func TestSubscriptionReport_NotConfigured(t *testing.T) {
	uc := NewSubscriptionReportUsecase(nil)
	_, err := uc.BuildReport(context.Background(), time.Time{}, time.Time{}, 0, 0, "")
	if err == nil {
		t.Fatal("expected error when repo is nil")
	}
}
