-- Subscription priority deduction: extend billing_reservations with the
-- subscription-side pre-deduction columns. All columns are nullable / default 0
-- so existing rows are untouched and the legacy balance-only path keeps
-- working. The new (user_id, status) index is what the absorber check uses to
-- aggregate active subscription pre-deductions when computing remaining quota.

ALTER TABLE `billing_reservations`
  ADD COLUMN `subscription_id` BIGINT NOT NULL DEFAULT 0,
  ADD COLUMN `subscription_amount_usd` DOUBLE NOT NULL DEFAULT 0,
  ADD COLUMN `subscription_daily_window_start` BIGINT NOT NULL DEFAULT 0,
  ADD COLUMN `subscription_weekly_window_start` BIGINT NOT NULL DEFAULT 0,
  ADD COLUMN `subscription_monthly_window_start` BIGINT NOT NULL DEFAULT 0,
  ADD COLUMN `balance_amount_quota` BIGINT NOT NULL DEFAULT 0;

CREATE INDEX `idx_billing_reservations_user_status` ON `billing_reservations`(`user_id`, `status`);
CREATE INDEX `idx_billing_reservations_subscription` ON `billing_reservations`(`subscription_id`, `status`);
