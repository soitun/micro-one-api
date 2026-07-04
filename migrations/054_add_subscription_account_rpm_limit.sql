-- Add per-account requests-per-minute limit for upstream subscription accounts.
ALTER TABLE `subscription_accounts`
  ADD COLUMN `rpm_limit` int NOT NULL DEFAULT 0 COMMENT 'local requests-per-minute limit for relay dispatch; 0 means unlimited';
