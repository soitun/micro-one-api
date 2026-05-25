-- Add a role column to users. Values follow the same scale as upstream
-- one-api so existing semantics carry over:
--   0   = guest
--   1   = common user (default)
--   10  = admin
--   100 = root
--
-- Existing deployments may have an admin user seeded by the previous core
-- baseline (username 'admin'). Promote that row to root so role-based admin
-- gates keep recognising it after this migration applies.
ALTER TABLE `users`
  ADD COLUMN `role` int NOT NULL DEFAULT 1 AFTER `status`,
  ADD KEY `idx_role` (`role`);

UPDATE `users` SET `role` = 100 WHERE `username` = 'admin' AND `role` < 100;
