package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/tabloy/keygate/internal/store"
)

// RenewalReminderService handles sending renewal reminders for expiring licenses
// Task 14.1: Create service struct
type RenewalReminderService struct {
	Store    *store.Store
	EmailSvc *EmailService
	Logger   *slog.Logger
}

// NewRenewalReminderService creates a new renewal reminder service
func NewRenewalReminderService(store *store.Store, email *EmailService, logger *slog.Logger) *RenewalReminderService {
	if logger == nil {
		logger = slog.Default()
	}

	return &RenewalReminderService{
		Store:    store,
		EmailSvc: email,
		Logger:   logger,
	}
}

// SendDueReminders sends renewal reminders for licenses at configured intervals
// Task 14.7: Loop through reminder intervals and send reminders
func (s *RenewalReminderService) SendDueReminders(ctx context.Context) error {
	// Task 14.10: Check if reminders are enabled
	enabled, err := s.Store.GetSetting(ctx, "renewal_reminders_enabled")
	if err != nil {
		s.Logger.Error("failed to check renewal reminders setting", "error", err)
		// Default to enabled if setting doesn't exist
	} else if enabled == "false" {
		s.Logger.Info("renewal reminders disabled, skipping")
		return nil
	}

	// Standard reminder intervals: 30, 14, 7, 1 days before expiration
	intervals := []int{30, 14, 7, 1}

	totalSent := 0
	for _, days := range intervals {
		sent, err := s.sendRemindersForInterval(ctx, days)
		if err != nil {
			s.Logger.Error("failed to send reminders for interval",
				"days", days,
				"error", err)
			// Continue with other intervals even if one fails
			continue
		}
		totalSent += sent
	}

	s.Logger.Info("renewal reminders batch complete",
		"total_sent", totalSent,
		"intervals", intervals)

	return nil
}

// sendRemindersForInterval sends reminders for a specific day interval
// Task 14.9: Batch processing to avoid overwhelming email queue
func (s *RenewalReminderService) sendRemindersForInterval(ctx context.Context, days int) (int, error) {
	const batchSize = 100 // Task 14.9: Process 100 licenses per batch

	// Task 14.2: Find licenses expiring in N days
	licenses, err := s.Store.FindLicensesExpiringIn(ctx, days)
	if err != nil {
		return 0, fmt.Errorf("failed to find expiring licenses: %w", err)
	}

	if len(licenses) == 0 {
		s.Logger.Debug("no licenses expiring in interval", "days", days)
		return 0, nil
	}

	s.Logger.Info("processing reminders for interval",
		"days", days,
		"total_licenses", len(licenses),
		"batches", (len(licenses)+batchSize-1)/batchSize)

	totalSent := 0

	// Task 14.9: Process in batches
	for i := 0; i < len(licenses); i += batchSize {
		end := i + batchSize
		if end > len(licenses) {
			end = len(licenses)
		}

		batch := licenses[i:end]
		batchNum := (i / batchSize) + 1

		s.Logger.Debug("processing batch",
			"batch", batchNum,
			"size", len(batch))

		sent := s.processBatch(ctx, batch, days)
		totalSent += sent
	}

	return totalSent, nil
}

// processBatch processes a batch of licenses
func (s *RenewalReminderService) processBatch(ctx context.Context, licenses []*store.ExpiringLicense, days int) int {
	sent := 0
	for _, license := range licenses {
		// Task 14.3: Check if reminder already sent
		alreadySent, err := s.Store.WasReminderSent(ctx, license.ID, days)
		if err != nil {
			s.Logger.Error("failed to check reminder status",
				"license_id", license.ID,
				"days", days,
				"error", err)
			continue
		}

		if alreadySent {
			s.Logger.Debug("reminder already sent",
				"license_id", license.ID,
				"days", days)
			continue
		}

		// Task 14.6: Send reminder email
		err = s.sendReminderEmail(ctx, license, days)
		if err != nil {
			s.Logger.Error("failed to send reminder email",
				"license_id", license.ID,
				"email", license.Email,
				"days", days,
				"error", err)
			continue
		}

		// Task 14.4: Mark reminder as sent
		err = s.Store.MarkReminderSent(ctx, license.ID, days)
		if err != nil {
			s.Logger.Error("failed to mark reminder as sent",
				"license_id", license.ID,
				"days", days,
				"error", err)
			// Don't continue - email was sent, so count it
		}

		sent++
		s.Logger.Info("renewal reminder sent",
			"license_id", license.ID,
			"email", license.Email,
			"days_remaining", days)
	}

	return sent
}

// sendReminderEmail sends a renewal reminder email to a license holder
// Task 14.6: Send email using template
func (s *RenewalReminderService) sendReminderEmail(ctx context.Context, license *store.ExpiringLicense, daysRemaining int) error {
	// Build renewal URL (TODO: Get base URL from config/settings)
	baseURL := "https://api.example.com" // Placeholder
	renewalURL := fmt.Sprintf("%s/portal/renew/%s", baseURL, license.ID)

	subject := fmt.Sprintf("Your %s license expires in %d days", license.ProductName, daysRemaining)

	// Simple text email for now
	// TODO: Task 14.5 - Implement HTML template rendering with renewal_reminder.html
	body := fmt.Sprintf(
		"Hello,\n\nYour license for %s (%s plan) expires in %d days on %s.\n\nRenew now: %s\n\nThank you!",
		license.ProductName,
		license.PlanName,
		daysRemaining,
		license.ValidUntil.Format("January 2, 2006"),
		renewalURL,
	)

	// Task 14.5: Use renewal_reminder template
	err := s.EmailSvc.Send(license.Email, subject, body)

	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
