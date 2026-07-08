-- Phase 2 review H10: enforce a single active subscription per user at the DB
-- level. A generated column exposes user_id only for active rows (NULL
-- otherwise) so a UNIQUE index on it permits multiple non-active rows per
-- user (expired/revoked) while forbidding two concurrent active rows.
-- Concurrent payment callbacks / new-purchase vs renewal races that both
-- read "no active" and CreateSubscription will now collide on this index
-- instead of producing two active subscriptions.

ALTER TABLE `user_subscriptions`
  ADD COLUMN `active_user_id` bigint GENERATED ALWAYS AS
    (IF(`status` = 'active', `user_id`, NULL)) VIRTUAL,
  ADD UNIQUE INDEX `uniq_user_subs_active_user_id` (`active_user_id`);
