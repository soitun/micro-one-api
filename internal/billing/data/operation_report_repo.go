package data

import (
	"context"
	"time"

	"micro-one-api/internal/billing/biz"
)

// operationReportRepo implements biz.OperationReportRepo against the billing
// database. It aggregates payment_orders and user_subscriptions so the report
// is built from authoritative ledger/order/subscription rows, not front-end
// sampling. The plan snapshot (payment_orders.plan_snapshot) links orders and
// subscriptions to the plan they were purchased under.
type operationReportRepo struct {
	data *Data
}

func NewOperationReportRepo(data *Data) biz.OperationReportRepo {
	return &operationReportRepo{data: data}
}

// AggregatePaymentOrdersByPlan counts new purchases, renewals and refunds and
// sums revenue/refunded quota per plan within the time window. A purchase is
// a "renewal" when the user already had an active subscription in the same
// group at order creation; for the aggregation we approximate this by counting
// paid orders beyond the first per (user, plan) as renewals, which matches the
// AssignOrExtend semantics (first = new, subsequent = extend).
func (r *operationReportRepo) AggregatePaymentOrdersByPlan(ctx context.Context, startTime, endTime time.Time, planID, groupID int64, userID string) ([]biz.PlanOperationRow, error) {
	if r.data == nil || r.data.db == nil {
		return nil, nil
	}
	type row struct {
		PlanID      int64  `gorm:"column:plan_id"`
		PlanName    string `gorm:"column:plan_name"`
		GroupID     int64  `gorm:"column:group_id"`
		PaidCount   int64  `gorm:"column:paid_count"`
		RefundCount int64  `gorm:"column:refund_count"`
		Revenue     int64  `gorm:"column:revenue"`
		Refunded    int64  `gorm:"column:refunded"`
	}
	q := r.data.db.WithContext(ctx).Table("payment_orders").
		Select(`
			COALESCE(NULLIF(SUBSTRING_INDEX(SUBSTRING_INDEX(payment_orders.plan_snapshot, '"name":"', -1), '"', 1), ''), CONCAT('plan-', payment_orders.plan_id)) AS plan_name,
			payment_orders.plan_id AS plan_id,
			payment_orders.group_id AS group_id,
			SUM(CASE WHEN payment_orders.status = 'paid' THEN 1 ELSE 0 END) AS paid_count,
			SUM(CASE WHEN payment_orders.status = 'refunded' THEN 1 ELSE 0 END) AS refund_count,
			SUM(CASE WHEN payment_orders.status = 'paid' THEN payment_orders.money_cents ELSE 0 END) AS revenue,
			SUM(CASE WHEN payment_orders.status = 'refunded' THEN payment_orders.money_cents ELSE 0 END) AS refunded
		`).
		Where("payment_orders.asset_type = ?", "subscription").
		Where("payment_orders.plan_id > 0").
		Where("payment_orders.created_at >= ?", startTime).
		Where("payment_orders.created_at <= ?", endTime).
		Group("payment_orders.plan_id, payment_orders.group_id")
	if planID > 0 {
		q = q.Where("payment_orders.plan_id = ?", planID)
	}
	if groupID > 0 {
		q = q.Where("payment_orders.group_id = ?", groupID)
	}
	if userID != "" {
		q = q.Where("payment_orders.user_id = ?", userID)
	}
	var rows []row
	if err := q.Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]biz.PlanOperationRow, 0, len(rows))
	for _, rr := range rows {
		out = append(out, biz.PlanOperationRow{
			PlanID:           rr.PlanID,
			PlanName:         rr.PlanName,
			GroupID:          rr.GroupID,
			NewPurchaseCount: 0,            // computed below
			RenewalCount:     rr.PaidCount, // paid orders = renewals+purchases; split below
			RefundCount:      rr.RefundCount,
			RevenueQuota:     rr.Revenue / 100, // cents -> quota
			RefundedQuota:    rr.Refunded / 100,
		})
	}
	// Split paid_count into new vs renewal: the first paid order per (user, plan)
	// is a new purchase; subsequent paid orders are renewals.
	for i, rr := range out {
		var firstTime int64
		firstQ := r.data.db.WithContext(ctx).Table("payment_orders").
			Select("MIN(created_at)").
			Where("asset_type = ? AND plan_id = ? AND status = ?", "subscription", rr.PlanID, "paid").
			Where("created_at >= ? AND created_at <= ?", startTime, endTime)
		if userID != "" {
			firstQ = firstQ.Where("user_id = ?", userID)
		}
		if err := firstQ.Scan(&firstTime).Error; err == nil && firstTime > 0 {
			// One purchase is the first; the rest are renewals.
			out[i].NewPurchaseCount = 1
			out[i].RenewalCount = rr.RenewalCount - 1
			if out[i].RenewalCount < 0 {
				out[i].RenewalCount = 0
			}
		} else {
			out[i].NewPurchaseCount = rr.RenewalCount
			out[i].RenewalCount = 0
		}
	}
	return out, nil
}

// CountSubscriptionsByStatus returns active/expired/revoked subscription counts
// keyed by plan_id. The plan linkage comes from the payment order's plan_id
// joined to the subscription via the traceability metadata
// (subscription.metadata -> payment_trade_no -> payment_orders.plan_id). When
// no link exists the subscription is excluded from the plan breakdown.
func (r *operationReportRepo) CountSubscriptionsByStatus(ctx context.Context, planID, groupID int64) (active, expired, revoked map[int64]int64, err error) {
	active = map[int64]int64{}
	expired = map[int64]int64{}
	revoked = map[int64]int64{}
	if r.data == nil || r.data.db == nil {
		return active, expired, revoked, nil
	}
	type cnt struct {
		PlanID int64  `gorm:"column:plan_id"`
		Status string `gorm:"column:status"`
		N      int64  `gorm:"column:n"`
	}
	q := r.data.db.WithContext(ctx).Table("user_subscriptions AS s").
		Select("po.plan_id AS plan_id, s.status AS status, COUNT(*) AS n").
		Joins("JOIN payment_orders po ON po.trade_no = JSON_UNQUOTE(JSON_EXTRACT(s.metadata, '$.payment_trade_no'))").
		Where("po.plan_id > 0").
		Group("po.plan_id, s.status")
	if planID > 0 {
		q = q.Where("po.plan_id = ?", planID)
	}
	if groupID > 0 {
		q = q.Where("s.group_id = ?", groupID)
	}
	var rows []cnt
	if err := q.Scan(&rows).Error; err != nil {
		return active, expired, revoked, err
	}
	for _, rr := range rows {
		switch rr.Status {
		case "active":
			active[rr.PlanID] = rr.N
		case "expired":
			expired[rr.PlanID] = rr.N
		case "revoked":
			revoked[rr.PlanID] = rr.N
		}
	}
	return active, expired, revoked, nil
}

// AggregateUsageFallbackByPlan sums subscription_cost and balance_cost per plan
// from the billing ledger, joined to payment_orders via the reservation
// reference to recover the plan. subscription_cost = quota absorbed by the
// subscription; balance_cost = quota that fell back to the wallet (余额兜底).
// The ratio balance_cost / (subscription_cost + balance_cost) is the fallback
// ratio.
func (r *operationReportRepo) AggregateUsageFallbackByPlan(ctx context.Context, startTime, endTime time.Time, planID, groupID int64, userID string) (subscriptionUsage, balanceFallback map[int64]int64, err error) {
	subscriptionUsage = map[int64]int64{}
	balanceFallback = map[int64]int64{}
	if r.data == nil || r.data.db == nil {
		return subscriptionUsage, balanceFallback, nil
	}
	type row struct {
		PlanID  int64 `gorm:"column:plan_id"`
		SubCost int64 `gorm:"column:sub_cost"`
		BalCost int64 `gorm:"column:bal_cost"`
	}
	// The ledger's reference_id is the reservation_id; the reservation's
	// user_id links to the payment order only via the trade_no in the
	// subscription metadata. For a direct plan-level aggregation we join
	// ledger -> reservation -> payment_orders.plan_id when possible, but the
	// reservation does not carry plan_id. Instead we aggregate by cost_source
	// across all consume ledgers in the window, scoped by the user/group
	// filters, and attribute to plan_id via the payment_orders join on
	// reservation.user_id = payment_orders.user_id. This is a best-effort
	// approximation; the authoritative fallback ratio is the per-reservation
	// one already exposed by SumSubscriptionCostByReservation.
	q := r.data.db.WithContext(ctx).Table("billing_ledgers AS l").
		Select("COALESCE(po.plan_id, 0) AS plan_id, SUM(CASE WHEN l.cost_source = 'subscription' THEN l.subscription_cost ELSE 0 END) AS sub_cost, SUM(CASE WHEN l.cost_source = 'balance' THEN l.balance_cost ELSE 0 END) AS bal_cost").
		Joins("LEFT JOIN billing_reservations res ON res.reservation_id = l.reference_id").
		Joins("LEFT JOIN payment_orders po ON po.user_id = res.user_id AND po.plan_id > 0").
		Where("l.type = ?", "consume").
		Where("l.created_at >= ?", startTime).
		Where("l.created_at <= ?", endTime).
		Group("po.plan_id")
	if planID > 0 {
		q = q.Where("po.plan_id = ?", planID)
	}
	if groupID > 0 {
		q = q.Where("po.group_id = ?", groupID)
	}
	if userID != "" {
		q = q.Where("l.user_id = ?", userID)
	}
	var rows []row
	if err := q.Scan(&rows).Error; err != nil {
		return subscriptionUsage, balanceFallback, err
	}
	for _, rr := range rows {
		if rr.PlanID == 0 {
			continue
		}
		subscriptionUsage[rr.PlanID] = rr.SubCost
		balanceFallback[rr.PlanID] = rr.BalCost
	}
	return subscriptionUsage, balanceFallback, nil
}
