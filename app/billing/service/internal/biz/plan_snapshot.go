package biz

import (
	"micro-one-api/domain/subscription/biz"
)

// PlanSnapshot is an alias to the canonical domain/subscription definition so
// billing-local code can refer to it without a second declaration. The struct
// itself, its encode/decode helpers and the reference-id helper all live in
// domain/subscription/biz so that admin, billing and relay share a single
// source of truth for the plan-snapshot contract.
type PlanSnapshot = biz.PlanSnapshot

// EncodePlanSnapshot delegates to domain/subscription/biz.EncodePlanSnapshot.
func EncodePlanSnapshot(s PlanSnapshot) string { return biz.EncodePlanSnapshot(s) }

// DecodePlanSnapshot delegates to domain/subscription/biz.DecodePlanSnapshot.
func DecodePlanSnapshot(raw string) (PlanSnapshot, error) {
	return biz.DecodePlanSnapshot(raw)
}

// PlanSnapshotReferenceID delegates to domain/subscription/biz.PlanSnapshotReferenceID.
func PlanSnapshotReferenceID(tradeNo string, planID int64) string {
	return biz.PlanSnapshotReferenceID(tradeNo, planID)
}
