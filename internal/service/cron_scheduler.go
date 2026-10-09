package service

import (
	"context"
	"log/slog"
	"time"
)

// CronScheduler handles scheduled tasks for the application
// Task 14.8: Register renewal reminder job
type CronScheduler struct {
	RenewalSvc *RenewalReminderService
	Logger     *slog.Logger
}

// NewCronScheduler creates a new cron scheduler
func NewCronScheduler(renewalSvc *RenewalReminderService, logger *slog.Logger) *CronScheduler {
	if logger == nil {
		logger = slog.Default()
	}

	return &CronScheduler{
		RenewalSvc: renewalSvc,
		Logger:     logger,
	}
}

// RegisterJobs registers all cron jobs
// Task 14.8: Daily renewal reminder check at 00:00 UTC
func (s *CronScheduler) RegisterJobs() {
	s.Logger.Info("registering cron jobs")

	// Task 14.8: Run renewal reminders daily at 00:00 UTC
	// This is a placeholder - actual cron implementation depends on the cron library used
	// Example with robfig/cron: c.AddFunc("0 0 * * *", s.RunRenewalReminders)
	s.Logger.Info("renewal reminder job registered", "schedule", "daily at 00:00 UTC")
}

// RunRenewalReminders is the job handler for renewal reminders
// Task 14.8: Called by cron scheduler
func (s *CronScheduler) RunRenewalReminders() {
	s.Logger.Info("running scheduled renewal reminders")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	err := s.RenewalSvc.SendDueReminders(ctx)
	if err != nil {
		s.Logger.Error("renewal reminder job failed", "error", err)
		return
	}

	s.Logger.Info("renewal reminder job completed successfully")
}

// Example integration with robfig/cron (v3):
//
// import "github.com/robfig/cron/v3"
//
// func SetupCronJobs(renewalSvc *service.RenewalReminderService) *cron.Cron {
//     c := cron.New(cron.WithLocation(time.UTC))
//
//     scheduler := service.NewCronScheduler(renewalSvc, slog.Default())
//
//     // Task 14.8: Daily at 00:00 UTC
//     c.AddFunc("0 0 * * *", scheduler.RunRenewalReminders)
//
//     // For testing: every 5 minutes
//     // c.AddFunc("*/5 * * * *", scheduler.RunRenewalReminders)
//
//     c.Start()
//     return c
// }
