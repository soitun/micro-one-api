-- Phase 2.3 traceability: persist the subscription_id granted by an order onto
-- the payment_orders row so refunds can deterministically resolve the exact
-- subscription to revoke/shorten (mirrors migrations/058_add_subscription_id_to_payment_orders.sql).

ALTER TABLE payment_orders
  ADD COLUMN IF NOT EXISTS subscription_id BIGINT NOT NULL DEFAULT 0;
