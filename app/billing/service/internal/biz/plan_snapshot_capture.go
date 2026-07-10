package biz

import (
	"context"
	"fmt"
)

// PlanSnapshotter captures the immutable purchase-time view of a plan into a
// PaymentOrder. The interface is kept separate from SubscriptionPlanGetter so
// the payment usecase can depend on a narrow capability instead of the full
// plan repo.
type PlanSnapshotter interface {
	CapturePlanSnapshot(ctx context.Context, planID int64) (PlanSnapshot, error)
}

// paymentPlanSnapshotter is the default implementation backed by the
// subscription plan repository. It reads the live plan row once and copies the
// fulfilment-relevant fields into a PlanSnapshot via the canonical
// SubscriptionPlan.ToPlanSnapshot helper (shared with the admin plan-snapshot
// completion path). The snapshot is then frozen on the payment order; later
// edits to the plan do not retroactively change the order.
type paymentPlanSnapshotter struct {
	plans SubscriptionPlanGetter
}

// NewPaymentPlanSnapshotter builds a snapshotter from a plan getter.
func NewPaymentPlanSnapshotter(plans SubscriptionPlanGetter) PlanSnapshotter {
	return &paymentPlanSnapshotter{plans: plans}
}

func (s *paymentPlanSnapshotter) CapturePlanSnapshot(ctx context.Context, planID int64) (PlanSnapshot, error) {
	if s == nil || s.plans == nil {
		return PlanSnapshot{}, fmt.Errorf("plan snapshotter is not configured")
	}
	if planID <= 0 {
		return PlanSnapshot{}, nil
	}
	plan, err := s.plans.GetPlanByID(ctx, planID)
	if err != nil {
		return PlanSnapshot{}, err
	}
	if plan == nil || plan.ID <= 0 {
		return PlanSnapshot{}, nil
	}
	return plan.ToPlanSnapshot(), nil
}

// ApplyPlanSnapshotToOrder encodes the snapshot onto the order's PlanSnapshot
// field. It is a no-op when the order has no plan_id or the snapshot is zero.
func ApplyPlanSnapshotToOrder(order *PaymentOrder, snapshot PlanSnapshot) {
	if order == nil {
		return
	}
	if snapshot.PlanID == 0 {
		return
	}
	order.PlanSnapshot = EncodePlanSnapshot(snapshot)
}
