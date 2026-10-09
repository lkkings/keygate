-- Add payment currency and renewal tracking columns to licenses table
ALTER TABLE licenses
    ADD COLUMN IF NOT EXISTS payment_currency TEXT,
    ADD COLUMN IF NOT EXISTS renewal_for_license_id TEXT REFERENCES licenses(id) ON DELETE SET NULL;

-- Add index for finding renewal payments
CREATE INDEX IF NOT EXISTS idx_licenses_renewal_for ON licenses(renewal_for_license_id);

-- Add comment to clarify usage
COMMENT ON COLUMN licenses.payment_currency IS 'Currency used for the payment that created this license (USD, CNY, HKD)';
COMMENT ON COLUMN licenses.renewal_for_license_id IS 'If this is a renewal payment, the original license being renewed';
