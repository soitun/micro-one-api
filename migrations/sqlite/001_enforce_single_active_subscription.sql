-- Phase 2 review H10: enforce a single active subscription per user at the DB
-- level via a partial unique index. SQLite supports partial indexes
-- (WHERE on CREATE INDEX), so we forbid two rows with status='active' for the
-- same user_id without a generated column.

CREATE UNIQUE INDEX IF NOT EXISTS uniq_user_subs_active_user_id
  ON user_subscriptions (user_id)
  WHERE status = 'active';
