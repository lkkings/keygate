-- Rollback additional indexes
DROP INDEX IF EXISTS idx_renewal_reminders_license_type;
DROP INDEX IF EXISTS idx_licenses_valid_until;
