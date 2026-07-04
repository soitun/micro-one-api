-- account_receivables is the append-only mirror of wallet overdraft events
-- produced by the dual-track commit pipeline. It exists so a recharge can be
-- matched back to the original reservation that drove the wallet negative,
-- without re-deriving the negative balance from the ledger.

CREATE TABLE IF NOT EXISTS `account_receivables` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` varchar(64) NOT NULL,
  `reservation_id` varchar(64) NOT NULL,
  `overdue_quota` bigint NOT NULL,
  `overdue_usd` double NOT NULL,
  `status` varchar(16) NOT NULL DEFAULT 'pending',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `settled_at` datetime(3) DEFAULT NULL,
  `settled_quota` bigint NOT NULL DEFAULT 0,
  `remark` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_account_receivable_reservation` (`reservation_id`),
  KEY `idx_account_receivable_user` (`user_id`, `status`),
  KEY `idx_account_receivable_status_created` (`status`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='钱包欠费应收明细（订阅优先结算镜像）';
