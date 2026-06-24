ALTER TABLE `notifications`
  ADD COLUMN `last_error` varchar(2048) NOT NULL DEFAULT '' AFTER `retry_count`;
