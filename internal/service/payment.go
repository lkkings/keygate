package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/tabloy/keygate/internal/model"
	"github.com/tabloy/keygate/internal/payment"
	"github.com/tabloy/keygate/internal/store"
)

// PaymentService handles payment operations across multiple providers
// NOTE: This is a stub implementation for initial multi-gateway work
type PaymentService struct {
	Store    *store.Store
	Webhook  *WebhookService
	EmailSvc *EmailService
	Logger   *slog.Logger
}

// NewPaymentService creates a new payment service
func NewPaymentService(store *store.Store, webhook *WebhookService, email *EmailService, logger *slog.Logger) *PaymentService {
	if logger == nil {
		logger = slog.Default()
	}

	return &PaymentService{
		Store:    store,
		Webhook:  webhook,
		EmailSvc: email,
		Logger:   logger,
	}
}

// CreateCheckout creates a checkout session for the given provider
func (s *PaymentService) CreateCheckout(ctx context.Context, providerName string, params payment.CheckoutParams) (payment.CheckoutResult, error) {
	provider, err := payment.GetProvider(providerName)
	if err != nil {
		s.Logger.Error("failed to get payment provider",
			"provider", providerName,
			"error", err)
		return payment.CheckoutResult{}, err
	}

	result, err := provider.CreateCheckout(ctx, params)
	if err != nil {
		s.Logger.Error("failed to create checkout",
			"provider", providerName,
			"error", err)
		return payment.CheckoutResult{}, err
	}

	s.Logger.Info("checkout created",
		"provider", providerName,
		"session_id", result.SessionID)

	return result, nil
}

// HandleWebhook processes a webhook from a payment provider
func (s *PaymentService) HandleWebhook(ctx context.Context, providerName string, payload []byte, signature string) error {
	provider, err := payment.GetProvider(providerName)
	if err != nil {
		s.Logger.Error("failed to get payment provider",
			"provider", providerName,
			"error", err)
		return err
	}

	event, err := provider.VerifyWebhook(ctx, payload, signature)
	if err != nil {
		s.Logger.Error("webhook verification failed",
			"provider", providerName,
			"error", err)
		return err
	}

	s.Logger.Info("webhook received",
		"provider", providerName,
		"event_type", event.EventType,
		"transaction_id", event.TransactionID)

	// Handle payment fulfillment
	switch event.EventType {
	case "payment.succeeded":
		return s.FulfillPayment(ctx, providerName, event)
	case "payment.refunded":
		return s.HandleRefund(ctx, providerName, event)
	default:
		s.Logger.Info("unhandled webhook event",
			"provider", providerName,
			"event_type", event.EventType)
		return nil
	}
}

// FulfillPayment creates or extends a license from a successful payment
// Task 13.5: Handle renewals by extending existing licenses
func (s *PaymentService) FulfillPayment(ctx context.Context, providerName string, event payment.WebhookEvent) error {
	// Check idempotency (Task 13.3 already implemented in transaction store)
	exists, err := s.Store.PaymentTransactionExists(ctx, providerName, event.TransactionID)
	if err != nil {
		s.Logger.Error("failed to check transaction existence",
			"provider", providerName,
			"transaction_id", event.TransactionID,
			"error", err)
		return err
	}

	if exists {
		s.Logger.Info("payment already fulfilled (idempotency)",
			"provider", providerName,
			"transaction_id", event.TransactionID)
		return nil
	}

	// Task 13.6: Audit log payment event
	s.Logger.Info("fulfilling payment",
		"provider", providerName,
		"transaction_id", event.TransactionID,
		"amount", event.Amount,
		"currency", event.Currency,
		"customer_email", event.CustomerEmail)

	// Task 13.5: Check if this is a renewal
	var licenseIDString string
	if event.RenewalForLicenseID != "" {
		// This is a renewal - extend existing license
		// Task 15.5: Calculate extension days based on plan
		extensionDays := event.ExtensionDays
		if extensionDays == 0 {
			// Default extension if not specified in event
			extensionDays = 365 // 1 year default
		}

		err := s.Store.ExtendLicense(ctx, event.RenewalForLicenseID, extensionDays)
		if err != nil {
			s.Logger.Error("failed to extend license",
				"license_id", event.RenewalForLicenseID,
				"error", err)
			return err
		}
		licenseIDString = event.RenewalForLicenseID

		s.Logger.Info("license renewed",
			"license_id", licenseIDString,
			"extension_days", extensionDays,
			"provider", providerName,
			"transaction_id", event.TransactionID)

		// Task 15.6: Clear renewal reminders for this license
		err = s.Store.ClearRenewalReminders(ctx, licenseIDString)
		if err != nil {
			// Log error but don't fail the renewal
			s.Logger.Error("failed to clear renewal reminders",
				"license_id", licenseIDString,
				"error", err)
		} else {
			s.Logger.Info("renewal reminders cleared",
				"license_id", licenseIDString)
		}
	} else {
		// New purchase - create license
		// TODO: Implement full license creation logic
		// For now, return error to indicate this needs implementation
		s.Logger.Error("license creation not yet implemented",
			"provider", providerName,
			"transaction_id", event.TransactionID)
		return fmt.Errorf("license creation not yet implemented")
	}

	// Create payment transaction record (Task 13.2, 13.4)
	err = s.Store.CreatePaymentTransaction(ctx, &model.PaymentTransaction{
		ProviderName:  providerName,
		ProviderTxID:  event.TransactionID,
		LicenseID:     licenseIDString,
		Amount:        event.Amount,
		Currency:      event.Currency,
		Status:        "completed",
		CustomerEmail: event.CustomerEmail,
		ProcessedAt:   time.Now(),
	})
	if err != nil {
		s.Logger.Error("failed to create payment transaction",
			"provider", providerName,
			"transaction_id", event.TransactionID,
			"error", err)
		return err
	}

	// Task 13.6: Audit log fulfillment completion
	s.Logger.Info("payment fulfilled",
		"provider", providerName,
		"transaction_id", event.TransactionID,
		"license_id", licenseIDString,
		"action", "payment_fulfilled")

	return nil
}

// HandleRefund processes a payment refund
// Task 13.6: Audit log refund events
func (s *PaymentService) HandleRefund(ctx context.Context, providerName string, event payment.WebhookEvent) error {
	s.Logger.Info("handling refund",
		"provider", providerName,
		"transaction_id", event.TransactionID,
		"amount", event.Amount,
		"action", "refund")

	// TODO: Implement refund handling (disable license, update transaction status)
	s.Logger.Warn("refund handling not yet implemented",
		"provider", providerName,
		"transaction_id", event.TransactionID)

	return nil
}

// RefundPayment initiates a refund for a payment
func (s *PaymentService) RefundPayment(ctx context.Context, transactionID string, amount int64, reason string) (payment.RefundResult, error) {
	// TODO: Lookup transaction, get provider, call RefundPayment
	return payment.RefundResult{}, fmt.Errorf("refund not yet implemented")
}

// GetPaymentStatus retrieves the status of a payment
func (s *PaymentService) GetPaymentStatus(ctx context.Context, providerName string, transactionID string) (payment.PaymentStatus, error) {
	provider, err := payment.GetProvider(providerName)
	if err != nil {
		return payment.PaymentStatus{}, err
	}

	return provider.GetPaymentStatus(ctx, transactionID)
}
