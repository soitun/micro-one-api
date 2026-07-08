-- Phase 2.3 traceability: persist the subscription_id granted by an order onto
-- the payment_orders row so refunds can deterministically resolve the exact
-- subscription to revoke/shorten. Previously the assigner wrote the link only
-- into subscription.metadata (payment_trade_no) and refunds had to fall back to
-- "the user's current active subscription", which is wrong when the user has
-- since purchased a new subscription. A dedicated column makes the order→
-- subscription link explicit and immutable for the refund path.

ALTER TABLE `payment_orders`
  ADD COLUMN `subscription_id` bigint NOT NULL DEFAULT 0 COMMENT 'subscription granted by this order (written at MarkOrderPaid)' AFTER `plan_id`;
