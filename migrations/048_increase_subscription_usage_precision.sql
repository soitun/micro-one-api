-- Keep small subscription-billed requests visible and cumulative.
-- The previous DECIMAL(12,4) rounded requests below 0.00005 USD to 0.0000.

ALTER TABLE `user_subscriptions`
  MODIFY COLUMN `daily_usage_usd` DECIMAL(18,8) NOT NULL DEFAULT 0,
  MODIFY COLUMN `weekly_usage_usd` DECIMAL(18,8) NOT NULL DEFAULT 0,
  MODIFY COLUMN `monthly_usage_usd` DECIMAL(18,8) NOT NULL DEFAULT 0;

