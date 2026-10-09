-- Rollback payment_transactions table
DROP INDEX IF EXISTS idx_payment_transactions_created_at;
DROP INDEX IF EXISTS idx_payment_transactions_provider_status;
DROP INDEX IF EXISTS idx_payment_transactions_license_id;
DROP TABLE IF EXISTS payment_transactions;
