-- Idempotency/event log for subscription-account local quota usage.
-- One successful billing commit should increment a local account budget at
-- most once for the same reservation/account/source tuple.

CREATE TABLE IF NOT EXISTS `subscription_account_quota_events` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `reservation_id` varchar(64) NOT NULL,
  `subscription_account_id` bigint NOT NULL,
  `cost_source` varchar(32) NOT NULL DEFAULT 'billing_commit',
  `cost_usd` decimal(18,6) NOT NULL DEFAULT 0 COMMENT 'raw committed cost in USD before account multiplier',
  `charged_usd` decimal(18,6) NOT NULL DEFAULT 0 COMMENT 'account-local cost after multiplier',
  `rate_multiplier` decimal(10,4) NOT NULL DEFAULT 1 COMMENT 'account multiplier snapshot used for charged_usd',
  `occurred_at` bigint NOT NULL DEFAULT 0,
  `created_at` bigint NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_subscription_account_quota_events_dedupe` (`reservation_id`, `subscription_account_id`, `cost_source`),
  KEY `idx_subscription_account_quota_events_account_time` (`subscription_account_id`, `occurred_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='subscription account local quota usage events';
