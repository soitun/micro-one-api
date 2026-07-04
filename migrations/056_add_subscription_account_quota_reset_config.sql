-- Add local quota reset strategy and timezone configuration for upstream subscription accounts.
ALTER TABLE `subscription_accounts`
  ADD COLUMN `quota_reset_strategy` varchar(16) NOT NULL DEFAULT 'rolling' COMMENT 'local quota reset strategy: rolling or fixed',
  ADD COLUMN `quota_timezone` varchar(64) NOT NULL DEFAULT 'UTC' COMMENT 'IANA timezone for fixed local quota reset windows';
