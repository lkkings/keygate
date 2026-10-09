-- Drop the complete Keygate schema created by 20261009_init.up.sql

DROP TABLE IF EXISTS activations, addons, analytics_snapshots, api_keys, 
audit_logs, email_queue, entitlements, floating_sessions, idempotency_keys, 
license_addons, license_renewals, licenses, metered_billing, notifications, 
oauth_accounts, otp_codes, payment_transactions, plan_update_terms, plans, 
processed_events, products, refresh_tokens, release_artifacts, 
release_signing_keys, releases, releases_legacy, renewal_reminders, seats, 
settings, subscriptions, usage_counters, usage_events, users, 
webhook_deliveries, webhooks CASCADE;

DROP FUNCTION IF EXISTS feed_gating_lock(text) CASCADE;
DROP FUNCTION IF EXISTS feed_gating_required(text) CASCADE;
DROP FUNCTION IF EXISTS licenses_init_updates_until() CASCADE;
DROP FUNCTION IF EXISTS licenses_require_gated_feed() CASCADE;
DROP FUNCTION IF EXISTS plans_prices_disjoint() CASCADE;
DROP FUNCTION IF EXISTS plans_record_update_terms() CASCADE;
DROP FUNCTION IF EXISTS plans_require_gated_feed() CASCADE;
DROP FUNCTION IF EXISTS products_keep_feed_gated() CASCADE;
