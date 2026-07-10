package biz

import (
	"testing"
)

func TestEncodeDecodePlanSnapshot_RoundTrip(t *testing.T) {
	in := PlanSnapshot{
		PlanID:       7,
		Name:         "Codex Pro Monthly",
		ProductName:  "codex-pro",
		GroupID:      3,
		PriceQuota:   2000,
		ValidityDays: 30,
		CapturedAt:   1700000000,
	}
	enc := EncodePlanSnapshot(in)
	if enc == "" {
		t.Fatal("encoded snapshot is empty for non-zero plan")
	}
	out, err := DecodePlanSnapshot(enc)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out != in {
		t.Fatalf("round trip mismatch:\n in  = %#v\n out = %#v", in, out)
	}
}

func TestEncodePlanSnapshot_ZeroIsEmpty(t *testing.T) {
	if got := EncodePlanSnapshot(PlanSnapshot{}); got != "" {
		t.Fatalf("zero snapshot encode = %q, want empty", got)
	}
}

func TestDecodePlanSnapshot_EmptyIsZero(t *testing.T) {
	out, err := DecodePlanSnapshot("")
	if err != nil {
		t.Fatalf("decode empty: %v", err)
	}
	if out.PlanID != 0 {
		t.Fatalf("empty decode plan_id = %d, want 0", out.PlanID)
	}
}

func TestDecodePlanSnapshot_Garbage(t *testing.T) {
	if _, err := DecodePlanSnapshot("not-json"); err == nil {
		t.Fatal("decode garbage: want error, got nil")
	}
}

func TestPlanSnapshotReferenceID_Stable(t *testing.T) {
	a := PlanSnapshotReferenceID("PAY-1", 7)
	b := PlanSnapshotReferenceID("PAY-1", 7)
	if a != b {
		t.Fatalf("reference id not stable: %q vs %q", a, b)
	}
	if PlanSnapshotReferenceID("PAY-1", 7) == PlanSnapshotReferenceID("PAY-1", 8) {
		t.Fatal("reference id collides across plan ids")
	}
	if PlanSnapshotReferenceID("PAY-1", 7) == PlanSnapshotReferenceID("PAY-2", 7) {
		t.Fatal("reference id collides across trade nos")
	}
}
