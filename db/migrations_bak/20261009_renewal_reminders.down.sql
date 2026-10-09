-- Rollback renewal_reminders table
DROP INDEX IF EXISTS idx_renewal_reminders_type;
DROP INDEX IF EXISTS idx_renewal_reminders_license_id;
DROP TABLE IF EXISTS renewal_reminders;
