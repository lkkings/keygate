package store

import (
	"context"
	"fmt"
	"time"
)

// ExpiringLicense represents a license that is expiring soon
type ExpiringLicense struct {
	ID          string
	Email       string
	PlanName    string
	ProductName string
	ValidUntil  time.Time
}

// FindLicensesExpiringIn finds licenses expiring in exactly N days
// Task 14.2: Query excluding perpetual licenses and Stripe subscriptions
func (s *Store) FindLicensesExpiringIn(ctx context.Context, days int) ([]*ExpiringLicense, error) {
	query := `
		SELECT
			l.id,
			l.email,
			p.name AS plan_name,
			pr.name AS product_name,
			l.valid_until
		FROM licenses l
		JOIN plans p ON l.plan_id = p.id
		JOIN products pr ON l.product_id = pr.id
		WHERE
			-- License expires in exactly N days (within a 24-hour window)
			l.valid_until >= now() + make_interval(days => ?)
			AND l.valid_until < now() + make_interval(days => ?)
			-- Exclude perpetual licenses (they don't expire)
			AND p.license_type != 'perpetual'
			-- Exclude Stripe subscriptions (they auto-renew)
			AND (l.stripe_subscription_id IS NULL OR l.stripe_subscription_id = '')
			-- Only active licenses
			AND l.status = 'active'
		ORDER BY l.valid_until ASC
	`

	rows, err := s.DB.QueryContext(ctx, query, days, days+1)
	if err != nil {
		return nil, fmt.Errorf("failed to query expiring licenses: %w", err)
	}
	defer rows.Close()

	var licenses []*ExpiringLicense
	for rows.Next() {
		var lic ExpiringLicense
		err := rows.Scan(
			&lic.ID,
			&lic.Email,
			&lic.PlanName,
			&lic.ProductName,
			&lic.ValidUntil,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan license: %w", err)
		}
		licenses = append(licenses, &lic)
	}

	return licenses, rows.Err()
}

// WasReminderSent checks if a renewal reminder was already sent for this license and interval
// Task 14.3: Check against renewal_reminders table
func (s *Store) WasReminderSent(ctx context.Context, licenseID string, days int) (bool, error) {
	query := `
		SELECT COUNT(*)
		FROM renewal_reminders
		WHERE license_id = ? AND days_before = ?
	`

	var count int
	err := s.DB.QueryRowContext(ctx, query, licenseID, days).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check reminder status: %w", err)
	}

	return count > 0, nil
}

// MarkReminderSent records that a renewal reminder was sent
// Task 14.4: Insert to renewal_reminders table
func (s *Store) MarkReminderSent(ctx context.Context, licenseID string, days int) error {
	query := `
		INSERT INTO renewal_reminders (license_id, days_before, sent_at)
		VALUES (?, ?, NOW())
	`

	_, err := s.DB.ExecContext(ctx, query, licenseID, days)
	if err != nil {
		return fmt.Errorf("failed to mark reminder as sent: %w", err)
	}

	return nil
}

// ClearRenewalReminders clears all reminder records for a license
// Task 15.6: Called after successful renewal to reset reminders for new period
func (s *Store) ClearRenewalReminders(ctx context.Context, licenseID string) error {
	query := `DELETE FROM renewal_reminders WHERE license_id = ?`

	_, err := s.DB.ExecContext(ctx, query, licenseID)
	if err != nil {
		return fmt.Errorf("failed to clear renewal reminders: %w", err)
	}

	return nil
}
