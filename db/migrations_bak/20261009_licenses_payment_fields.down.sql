-- Rollback payment currency and renewal tracking columns from licenses table
DROP INDEX IF EXISTS idx_licenses_renewal_for;
ALTER TABLE licenses
    DROP COLUMN IF EXISTS renewal_for_license_id,
    DROP COLUMN IF EXISTS payment_currency;
