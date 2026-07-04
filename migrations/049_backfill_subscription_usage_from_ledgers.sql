-- Recalculate active subscription usage from committed subscription ledgers.
-- This repairs rows that were committed before subscription usage precision
-- was increased or while commit retries were failing before ledger writes.

UPDATE `user_subscriptions` us
SET
  `daily_usage_usd` = (
    SELECT COALESCE(SUM(bl.`subscription_cost`), 0) / 10000
    FROM `billing_ledgers` bl
    JOIN `billing_reservations` br ON br.`reservation_id` = bl.`reference_id`
    WHERE br.`subscription_id` = us.`id`
      AND br.`status` = 'committed'
      AND br.`subscription_daily_window_start` = us.`daily_window_start`
      AND bl.`type` = 'consume'
      AND bl.`cost_source` = 'subscription'
  ),
  `weekly_usage_usd` = (
    SELECT COALESCE(SUM(bl.`subscription_cost`), 0) / 10000
    FROM `billing_ledgers` bl
    JOIN `billing_reservations` br ON br.`reservation_id` = bl.`reference_id`
    WHERE br.`subscription_id` = us.`id`
      AND br.`status` = 'committed'
      AND br.`subscription_weekly_window_start` = us.`weekly_window_start`
      AND bl.`type` = 'consume'
      AND bl.`cost_source` = 'subscription'
  ),
  `monthly_usage_usd` = (
    SELECT COALESCE(SUM(bl.`subscription_cost`), 0) / 10000
    FROM `billing_ledgers` bl
    JOIN `billing_reservations` br ON br.`reservation_id` = bl.`reference_id`
    WHERE br.`subscription_id` = us.`id`
      AND br.`status` = 'committed'
      AND br.`subscription_monthly_window_start` = us.`monthly_window_start`
      AND bl.`type` = 'consume'
      AND bl.`cost_source` = 'subscription'
  ),
  `updated_at` = UNIX_TIMESTAMP()
WHERE us.`status` = 'active';
