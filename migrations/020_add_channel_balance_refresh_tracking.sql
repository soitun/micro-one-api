-- Track per-channel balance refresh outcomes so admin can see last success/error,
-- and so the auto-disable threshold can react to persistent upstream failures.

ALTER TABLE `channels`
  ADD COLUMN `balance_refresh_last_error` text AFTER `balance_updated_time`,
  ADD COLUMN `balance_refresh_last_success_time` bigint DEFAULT 0 AFTER `balance_refresh_last_error`,
  ADD COLUMN `consecutive_balance_refresh_failures` int DEFAULT 0 AFTER `balance_refresh_last_success_time`;
