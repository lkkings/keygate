-- Add additional indexes for performance optimization

-- Index for finding licenses by expiration date (renewal reminder queries)
CREATE INDEX IF NOT EXISTS idx_licenses_valid_until ON licenses(valid_until) WHERE valid_until IS NOT NULL;

-- Index for renewal reminder queries by license and type
CREATE INDEX IF NOT EXISTS idx_renewal_reminders_license_type ON renewal_reminders(license_id, reminder_type);

-- Note: idx_payment_transactions_license_id and idx_payment_transactions_provider_status
-- were already created in 20261009_payment_transactions.up.sql
