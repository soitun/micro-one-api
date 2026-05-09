ALTER TABLE `tokens`
  ADD COLUMN `name` varchar(128) DEFAULT '' AFTER `user_id`,
  ADD COLUMN `created_at` bigint DEFAULT 0 AFTER `models`;
