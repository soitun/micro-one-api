-- Add pricing + duration to subscription groups so users can self-purchase a
-- subscription with their wallet balance. Both default 0, meaning "not
-- purchasable" (admin-assign only) — existing groups stay admin-only until set.

ALTER TABLE `subscription_groups`
  ADD COLUMN `price_quota` BIGINT NOT NULL DEFAULT 0,
  ADD COLUMN `duration_days` INT NOT NULL DEFAULT 0;
