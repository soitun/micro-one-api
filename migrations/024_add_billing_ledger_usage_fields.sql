ALTER TABLE `billing_ledgers`
  ADD COLUMN `token_name` varchar(128) DEFAULT '' AFTER `remark`,
  ADD COLUMN `model_name` varchar(128) DEFAULT '' AFTER `token_name`,
  ADD COLUMN `quota` bigint DEFAULT 0 AFTER `model_name`,
  ADD COLUMN `prompt_tokens` bigint DEFAULT 0 AFTER `quota`,
  ADD COLUMN `completion_tokens` bigint DEFAULT 0 AFTER `prompt_tokens`,
  ADD COLUMN `channel_id` bigint DEFAULT 0 AFTER `completion_tokens`,
  ADD COLUMN `elapsed_time` bigint DEFAULT 0 AFTER `channel_id`,
  ADD COLUMN `is_stream` tinyint(1) DEFAULT 0 AFTER `elapsed_time`,
  ADD COLUMN `endpoint` varchar(128) DEFAULT '' AFTER `is_stream`,
  ADD KEY `idx_billing_ledgers_user_created_model` (`user_id`, `created_at`, `model_name`);
