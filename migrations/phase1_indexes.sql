-- Phase 1: Database Index Optimization
-- Based on ARCHITECTURE_REFACTOR.md §5.2
--
-- This migration adds critical indexes for improved query performance.
-- Execute each statement individually and monitor impact.

-- ============================================================================
-- logs table indexes
-- ============================================================================

-- Index for user log queries
ALTER TABLE logs
ADD INDEX idx_user_id (user_id);

-- Composite index for user log pagination (most common query pattern)
ALTER TABLE logs
ADD INDEX idx_user_id_created_at (user_id, created_at);

-- Index for request tracing
ALTER TABLE logs
ADD INDEX idx_request_id (request_id(32));

-- ============================================================================
-- billing_ledgers table indexes
-- ============================================================================

-- Composite index for user consumption records (common query pattern)
ALTER TABLE billing_ledgers
ADD INDEX idx_user_id_created_at (user_id, created_at);

-- Composite index for channel usage statistics
ALTER TABLE billing_ledgers
ADD INDEX idx_channel_id_created_at (channel_id, created_at);

-- Index for model-based cost analysis
ALTER TABLE billing_ledgers
ADD INDEX idx_model_name_created_at (model_name(64), created_at);

-- Time-series index idx_created_at is created by 009_create_billing_ledgers.sql.

-- ============================================================================
-- channels table indexes
-- ============================================================================

-- Composite index for channel filtering by status and group
ALTER TABLE channels
ADD INDEX idx_group_status (status, `group`);

-- Index for channel selection with priority ordering
ALTER TABLE channels
ADD INDEX idx_group_status_priority (status, `group`, priority DESC);

-- ============================================================================
-- tokens table indexes
-- ============================================================================

-- Composite index for valid token lookups
ALTER TABLE tokens
ADD INDEX idx_status_expired (status, expired_time);

-- ============================================================================
-- billing_reservations table indexes
-- ============================================================================

-- The user/status index is created by 044_add_reservation_subscription_fields.sql.

-- Index for expired reservation cleanup
ALTER TABLE billing_reservations
ADD INDEX idx_expired_at (expired_at);

-- ============================================================================
-- Verification queries
-- ============================================================================
--
-- NOTE: The migration runner splits this file on ';' and executes every
-- statement as migration DDL. SHOW INDEX statements return result sets and are
-- not schema changes, so they must not live here — run them manually against a
-- MySQL client if you want to verify index creation:
--   SHOW INDEX FROM logs WHERE Key_name LIKE 'idx_%';
--   SHOW INDEX FROM billing_ledgers WHERE Key_name LIKE 'idx_%';
--   SHOW INDEX FROM channels WHERE Key_name LIKE 'idx_%';
--   SHOW INDEX FROM tokens WHERE Key_name LIKE 'idx_%';
--   SHOW INDEX FROM billing_reservations WHERE Key_name LIKE 'idx_%';

-- ============================================================================
-- Rollback commands (if needed)
-- ============================================================================

-- ALTER TABLE logs DROP INDEX idx_user_id;
-- ALTER TABLE logs DROP INDEX idx_user_id_created_at;
-- ALTER TABLE logs DROP INDEX idx_request_id;
-- ALTER TABLE billing_ledgers DROP INDEX idx_user_id_created_at;
-- ALTER TABLE billing_ledgers DROP INDEX idx_channel_id_created_at;
-- ALTER TABLE billing_ledgers DROP INDEX idx_model_created_at;
-- ALTER TABLE billing_ledgers DROP INDEX idx_created_at;
-- ALTER TABLE channels DROP INDEX idx_group_status;
-- ALTER TABLE channels DROP INDEX idx_group_status_priority;
-- ALTER TABLE tokens DROP INDEX idx_status_expired;
-- ALTER TABLE billing_reservations DROP INDEX idx_expired_at;
