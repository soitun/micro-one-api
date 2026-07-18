-- ────────────────────────────────────────────────────────────────────────────
-- Phase 2.4 — Per-service schema isolation.
--
-- This file is NOT applied automatically. It is a reference DDL that an
-- operator runs once against an existing MySQL instance to split the
-- shared `oneapi` database into one schema per backend service:
--
--   oneapi_identity, oneapi_channel, oneapi_billing, oneapi_log,
--   oneapi_admin, oneapi_config, oneapi_notify, oneapi_monitor
--
-- Each backend service is then pointed at its own schema via the
-- <SVC>_SCHEMA environment variable (see docs/deployment.md §10). Tables
-- are copied (CREATE TABLE ... LIKE + INSERT ... SELECT) so the source
-- database is left untouched — a rollback is simply re-pointing the env
-- var back at `oneapi`.
--
-- Run with:
--   mysql -h <host> -u root -p < migrations/schema_split.sql
--
-- Then run the per-service migrations to bring each schema up to date:
--   MIGRATIONS_DSN='root:pw@tcp(host:3306)/oneapi_identity' \
--     go run ./cmd/migrate -ownership identity
--   MIGRATIONS_DSN='root:pw@tcp(host:3306)/oneapi_billing' \
--     go run ./cmd/migrate -ownership billing
--   …
-- ────────────────────────────────────────────────────────────────────────────

CREATE DATABASE IF NOT EXISTS oneapi_identity   DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS oneapi_channel    DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS oneapi_billing    DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS oneapi_log        DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS oneapi_admin      DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS oneapi_config     DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS oneapi_notify     DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS oneapi_monitor    DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- identity schema: users, tokens, user_oauth_identities, user_aff_records
CREATE TABLE IF NOT EXISTS oneapi_identity.users                  LIKE oneapi.users;
CREATE TABLE IF NOT EXISTS oneapi_identity.tokens                 LIKE oneapi.tokens;
CREATE TABLE IF NOT EXISTS oneapi_identity.user_oauth_identities  LIKE oneapi.user_oauth_identities;
INSERT IGNORE INTO oneapi_identity.users                     SELECT * FROM oneapi.users;
INSERT IGNORE INTO oneapi_identity.tokens                    SELECT * FROM oneapi.tokens;
INSERT IGNORE INTO oneapi_identity.user_oauth_identities     SELECT * FROM oneapi.user_oauth_identities;

-- channel schema: channels, abilities, subscription_accounts,
--                 subscription_account_quota_events, subscription_account_quota_reset_runs
CREATE TABLE IF NOT EXISTS oneapi_channel.channels                                   LIKE oneapi.channels;
CREATE TABLE IF NOT EXISTS oneapi_channel.abilities                                  LIKE oneapi.abilities;
CREATE TABLE IF NOT EXISTS oneapi_channel.subscription_accounts                      LIKE oneapi.subscription_accounts;
CREATE TABLE IF NOT EXISTS oneapi_channel.subscription_account_quota_events          LIKE oneapi.subscription_account_quota_events;
CREATE TABLE IF NOT EXISTS oneapi_channel.subscription_account_quota_reset_runs      LIKE oneapi.subscription_account_quota_reset_runs;
INSERT IGNORE INTO oneapi_channel.channels                                  SELECT * FROM oneapi.channels;
INSERT IGNORE INTO oneapi_channel.abilities                                 SELECT * FROM oneapi.abilities;
INSERT IGNORE INTO oneapi_channel.subscription_accounts                     SELECT * FROM oneapi.subscription_accounts;
INSERT IGNORE INTO oneapi_channel.subscription_account_quota_events         SELECT * FROM oneapi.subscription_account_quota_events;
INSERT IGNORE INTO oneapi_channel.subscription_account_quota_reset_runs     SELECT * FROM oneapi.subscription_account_quota_reset_runs;

-- billing schema: billing_*, payment_orders, reconciliation_runs,
--                 account_receivables, user_subscriptions, subscription_groups,
--                 subscription_plans, account_quota_snapshots
CREATE TABLE IF NOT EXISTS oneapi_billing.billing_reservations                 LIKE oneapi.billing_reservations;
CREATE TABLE IF NOT EXISTS oneapi_billing.billing_ledgers                      LIKE oneapi.billing_ledgers;
CREATE TABLE IF NOT EXISTS oneapi_billing.billing_redeem_codes                 LIKE oneapi.billing_redeem_codes;
CREATE TABLE IF NOT EXISTS oneapi_billing.billing_redeem_records               LIKE oneapi.billing_redeem_records;
CREATE TABLE IF NOT EXISTS oneapi_billing.payment_orders                       LIKE oneapi.payment_orders;
CREATE TABLE IF NOT EXISTS oneapi_billing.reconciliation_runs                  LIKE oneapi.reconciliation_runs;
CREATE TABLE IF NOT EXISTS oneapi_billing.account_receivables                  LIKE oneapi.account_receivables;
CREATE TABLE IF NOT EXISTS oneapi_billing.user_subscriptions                   LIKE oneapi.user_subscriptions;
CREATE TABLE IF NOT EXISTS oneapi_billing.subscription_groups                  LIKE oneapi.subscription_groups;
CREATE TABLE IF NOT EXISTS oneapi_billing.subscription_plans                   LIKE oneapi.subscription_plans;
CREATE TABLE IF NOT EXISTS oneapi_billing.account_quota_snapshots              LIKE oneapi.account_quota_snapshots;
INSERT IGNORE INTO oneapi_billing.billing_reservations        SELECT * FROM oneapi.billing_reservations;
INSERT IGNORE INTO oneapi_billing.billing_ledgers             SELECT * FROM oneapi.billing_ledgers;
INSERT IGNORE INTO oneapi_billing.billing_redeem_codes        SELECT * FROM oneapi.billing_redeem_codes;
INSERT IGNORE INTO oneapi_billing.billing_redeem_records      SELECT * FROM oneapi.billing_redeem_records;
INSERT IGNORE INTO oneapi_billing.payment_orders              SELECT * FROM oneapi.payment_orders;
INSERT IGNORE INTO oneapi_billing.reconciliation_runs         SELECT * FROM oneapi.reconciliation_runs;
INSERT IGNORE INTO oneapi_billing.account_receivables         SELECT * FROM oneapi.account_receivables;
INSERT IGNORE INTO oneapi_billing.user_subscriptions          SELECT * FROM oneapi.user_subscriptions;
INSERT IGNORE INTO oneapi_billing.subscription_groups         SELECT * FROM oneapi.subscription_groups;
INSERT IGNORE INTO oneapi_billing.subscription_plans          SELECT * FROM oneapi.subscription_plans;
INSERT IGNORE INTO oneapi_billing.account_quota_snapshots     SELECT * FROM oneapi.account_quota_snapshots;

-- log schema: logs
CREATE TABLE IF NOT EXISTS oneapi_log.logs LIKE oneapi.logs;
-- logs is typically high-volume; copy only recent rows to keep the cutover fast.
INSERT IGNORE INTO oneapi_log.logs SELECT * FROM oneapi.logs WHERE created_at >= UNIX_TIMESTAMP() - 7*24*3600;

-- admin schema: system_options
CREATE TABLE IF NOT EXISTS oneapi_admin.system_options LIKE oneapi.system_options;
INSERT IGNORE INTO oneapi_admin.system_options SELECT * FROM oneapi.system_options;

-- config schema: configs
CREATE TABLE IF NOT EXISTS oneapi_config.configs LIKE oneapi.configs;
INSERT IGNORE INTO oneapi_config.configs SELECT * FROM oneapi.configs;

-- notify schema: notifications
CREATE TABLE IF NOT EXISTS oneapi_notify.notifications LIKE oneapi.notifications;
INSERT IGNORE INTO oneapi_notify.notifications SELECT * FROM oneapi.notifications;

-- monitor schema: health_checks, alert_rules
CREATE TABLE IF NOT EXISTS oneapi_monitor.health_checks LIKE oneapi.health_checks;
CREATE TABLE IF NOT EXISTS oneapi_monitor.alert_rules   LIKE oneapi.alert_rules;
INSERT IGNORE INTO oneapi_monitor.health_checks SELECT * FROM oneapi.health_checks;
INSERT IGNORE INTO oneapi_monitor.alert_rules   SELECT * FROM oneapi.alert_rules;

-- NOTE: billing-ledger reconciliation reads from channels/logs/user_subscriptions
-- (see app/billing/internal/data/reconciliation_repo.go). When the billing
-- service is moved to oneapi_billing, those reads must either (a) keep a
-- cross-schema view in oneapi_billing pointing at the source tables, or
-- (b) move the read-side queries to a service boundary. Option (a) is the
-- transitional path documented in docs/deployment.md §10.4.
