-- Add frozen_quota column to users table
ALTER TABLE `users`
ADD COLUMN `frozen_quota` bigint DEFAULT 0 COMMENT '冻结配额（预扣中）' AFTER `quota`;
