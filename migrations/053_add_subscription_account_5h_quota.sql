-- Add a local rolling 5h budget window for upstream subscription accounts.
ALTER TABLE `subscription_accounts`
  ADD COLUMN `quota_5h_limit_usd` decimal(18,6) NOT NULL DEFAULT 0 COMMENT 'local rolling 5h budget limit in USD; 0 means unlimited',
  ADD COLUMN `quota_5h_used_usd` decimal(18,6) NOT NULL DEFAULT 0 COMMENT 'local rolling 5h budget used in USD',
  ADD COLUMN `quota_5h_window_start` bigint NOT NULL DEFAULT 0 COMMENT 'unix ts when the local 5h window started';

DROP INDEX `idx_subscription_local_quota` ON `subscription_accounts`;
CREATE INDEX `idx_subscription_local_quota` ON `subscription_accounts` (`status`, `quota_5h_window_start`, `quota_daily_window_start`, `quota_weekly_window_start`);
