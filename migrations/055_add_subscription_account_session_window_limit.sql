-- Add per-session cost window limit for upstream subscription accounts.
ALTER TABLE `subscription_accounts`
  ADD COLUMN `session_window_limit_usd` decimal(18,6) NOT NULL DEFAULT 0 COMMENT 'local per session_hash cost limit in USD; 0 means unlimited';
