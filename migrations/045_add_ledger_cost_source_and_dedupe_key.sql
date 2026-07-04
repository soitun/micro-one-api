-- Subscription priority deduction: extend billing_ledgers with cost-source
-- dimensioning and a unique idempotency key. The dedupe key is the new
-- authoritative uniqueness anchor for the commit pipeline; reference_id
-- remains for query convenience only.

ALTER TABLE `billing_ledgers`
  ADD COLUMN `cost_source` VARCHAR(16) NOT NULL DEFAULT 'balance',
  ADD COLUMN `subscription_cost` BIGINT NOT NULL DEFAULT 0,
  ADD COLUMN `balance_cost` BIGINT NOT NULL DEFAULT 0,
  ADD COLUMN `ledger_dedupe_key` VARCHAR(160) NOT NULL DEFAULT '';

-- Backfill dedupe keys for legacy entries so the unique index can be created
-- without violating constraints on existing rows. Legacy entries use the
-- same key format (reference_id:type) so the dedupe key still distinguishes
-- them at query time.
UPDATE `billing_ledgers`
  SET `ledger_dedupe_key` = CONCAT(IFNULL(`reference_id`, ''), ':', `type`, ':legacy')
  WHERE `ledger_dedupe_key` = '';

CREATE UNIQUE INDEX `idx_ledger_dedupe_key` ON `billing_ledgers`(`ledger_dedupe_key`);

CREATE INDEX `idx_billing_ledgers_cost_source_created` ON `billing_ledgers`(`cost_source`, `created_at`);
