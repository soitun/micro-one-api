-- Add group_id field to payment_orders for subscription purchase
-- This allows linking a payment order to a subscription group.
-- When payment is completed, the subscription will be automatically assigned.

ALTER TABLE payment_orders ADD COLUMN IF NOT EXISTS group_id BIGINT DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_payment_orders_group_id ON payment_orders(group_id);
