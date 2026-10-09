-- Drop all tables in reverse dependency order

DROP TABLE IF EXISTS processed_events;
DROP TABLE IF EXISTS settings;
DROP TABLE IF EXISTS otp_codes;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS release_signing_keys;
DROP TABLE IF EXISTS release_artifacts;
DROP TABLE IF EXISTS releases;
DROP TABLE IF EXISTS license_addons;
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS renewal_reminders;
DROP TABLE IF EXISTS email_queue;
DROP TABLE IF EXISTS payment_transactions;
DROP TABLE IF EXISTS activations;
DROP TABLE IF EXISTS licenses;
DROP TABLE IF EXISTS plans;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS users;
