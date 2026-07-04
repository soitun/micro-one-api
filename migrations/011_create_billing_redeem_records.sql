-- Create billing_redeem_records table
CREATE TABLE IF NOT EXISTS `billing_redeem_records` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` varchar(64) NOT NULL COMMENT '用户 ID',
  `code` varchar(64) NOT NULL COMMENT '兑换码',
  `amount` bigint NOT NULL COMMENT '兑换金额',
  `balance_before` bigint NOT NULL COMMENT '兑换前余额',
  `balance_after` bigint NOT NULL COMMENT '兑换后余额',
  `created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_code` (`code`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='兑换记录表';
