ALTER TABLE `reconciliation_runs`
  ADD COLUMN `total_channels` int NOT NULL DEFAULT 0 AFTER `total_accounts`;
