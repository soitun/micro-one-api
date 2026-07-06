-- Phase 2 subscription productization: capture a plan snapshot at payment order
-- creation so fulfillment is decoupled from later on/off-shelf changes to the
-- subscription_plan row. The snapshot stores the plan's immutable purchase-time
-- attributes (name, price, validity, group_id) so MarkOrderPaid can issue the
-- subscription even if the plan is later marked for_sale=false or deleted.

ALTER TABLE `payment_orders`
  ADD COLUMN `plan_snapshot` text NULL AFTER `plan_id`;
