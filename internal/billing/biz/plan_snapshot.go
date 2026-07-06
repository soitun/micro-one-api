package biz

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// PlanSnapshot captures the immutable purchase-time attributes of a
// subscription plan. It is stored on the payment order at creation time so
// MarkOrderPaid can issue the subscription even if the plan is later marked
// for_sale=false, edited, or deleted. Without the snapshot a closed-but-unpaid
// order could be stranded (cannot fulfill from a now-off-shelf plan) or, worse,
// fulfilled from a plan whose price/validity was changed mid-flight.
//
// The snapshot deliberately stores primitive copies of the fields the assigner
// needs (Name, GroupID, PriceQuota, ValidityDays) instead of a reference to the
// plan id, because the assigner must not re-read the live plan row at
// fulfillment time.
type PlanSnapshot struct {
	PlanID       int64  `json:"plan_id"`
	Name         string `json:"name"`
	ProductName  string `json:"product_name"`
	GroupID      int64  `json:"group_id"`
	PriceQuota   int64  `json:"price_quota"`
	ValidityDays int32  `json:"validity_days"`
	CapturedAt   int64  `json:"captured_at"`
}

// EncodePlanSnapshot serialises a snapshot into the JSON string persisted in
// payment_orders.plan_snapshot. An empty (zero-value) snapshot is encoded as
// the empty string so the column stays NULL for non-plan orders.
func EncodePlanSnapshot(s PlanSnapshot) string {
	if s.PlanID == 0 {
		return ""
	}
	if s.CapturedAt == 0 {
		s.CapturedAt = time.Now().Unix()
	}
	b, err := json.Marshal(s)
	if err != nil {
		return ""
	}
	return string(b)
}

// DecodePlanSnapshot parses the plan_snapshot column. An empty/NULL value is
// returned as the zero snapshot with a nil error, so callers can treat
// pre-snapshot orders uniformly.
func DecodePlanSnapshot(raw string) (PlanSnapshot, error) {
	if raw == "" {
		return PlanSnapshot{}, nil
	}
	var s PlanSnapshot
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		return PlanSnapshot{}, fmt.Errorf("decode plan_snapshot: %w", err)
	}
	return s, nil
}

// PlanSnapshotReferenceID returns the idempotency-friendly reference id the
// assigner records against the ledger when it fulfills a snapshot-backed
// order. It is stable across payment retries of the same trade_no.
func PlanSnapshotReferenceID(tradeNo string, planID int64) string {
	return fmt.Sprintf("payment:%s:plan:%s", tradeNo, strconv.FormatInt(planID, 10))
}
