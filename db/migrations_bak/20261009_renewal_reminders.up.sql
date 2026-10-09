-- Create renewal_reminders table for tracking sent renewal notifications
CREATE TABLE renewal_reminders (
    id TEXT PRIMARY KEY,
    license_id TEXT NOT NULL REFERENCES licenses(id) ON DELETE CASCADE,
    reminder_type TEXT NOT NULL CHECK (reminder_type IN ('30d', '14d', '7d', '1d')),
    sent_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_license_reminder_type UNIQUE (license_id, reminder_type)
);

-- Index for querying reminders by license
CREATE INDEX idx_renewal_reminders_license_id ON renewal_reminders(license_id);

-- Index for querying by reminder type
CREATE INDEX idx_renewal_reminders_type ON renewal_reminders(reminder_type);

-- Add comment
COMMENT ON TABLE renewal_reminders IS 'Tracks which renewal reminder emails have been sent to prevent duplicates';
