-- Ensure ledger rows written after 045 but before the service-side default
-- dedupe-key fix no longer reserve the empty unique key value.

UPDATE `billing_ledgers`
SET `ledger_dedupe_key` = CONCAT(
  CASE
    WHEN `reference_id` IS NULL OR `reference_id` = '' THEN CONCAT('legacy:', `user_id`, ':', `id`)
    ELSE `reference_id`
  END,
  ':',
  `type`,
  ':legacy'
)
WHERE `ledger_dedupe_key` = '';

UPDATE `billing_ledgers`
SET `cost_source` = 'balance',
    `balance_cost` = ABS(`amount`)
WHERE `type` = 'consume'
  AND (`cost_source` = '' OR `cost_source` = 'balance')
  AND `subscription_cost` = 0
  AND `balance_cost` = 0;
