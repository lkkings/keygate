package payment

import (
	"context"
	"time"
)

// PaymentProvider defines the interface that all payment providers must implement
type PaymentProvider interface {
	// Name returns the unique identifier for this provider (e.g., "stripe", "alipay", "wechat", "epay")
	Name() string

	// CreateCheckout creates a payment session and returns the checkout URL, session ID, and any error
	CreateCheckout(ctx context.Context, params CheckoutParams) (CheckoutResult, error)

	// VerifyWebhook verifies the authenticity of a webhook request and returns the parsed event
	VerifyWebhook(ctx context.Context, payload []byte, signature string) (WebhookEvent, error)

	// RefundPayment initiates a refund for a completed payment
	RefundPayment(ctx context.Context, transactionID string, amount int64, reason string) (RefundResult, error)

	// GetPaymentStatus retrieves the current status of a payment
	GetPaymentStatus(ctx context.Context, transactionID string) (PaymentStatus, error)
}

// CheckoutParams contains parameters for creating a checkout session
type CheckoutParams struct {
	PlanID        string
	ProductID     string
	CustomerEmail string
	Currency      string // "usd", "cny", "hkd"
	Amount        int64  // Amount in smallest currency unit (cents)
	SuccessURL    string
	CancelURL     string
	Mode          string // "payment" or "subscription"
	RenewalFor    string // License ID if this is a renewal payment
	Metadata      map[string]string
}

// CheckoutResult contains the result of creating a checkout session
type CheckoutResult struct {
	CheckoutURL string // URL to redirect user to
	SessionID   string // Provider-specific session identifier
	QRCodeURL   string // Optional: For QR code based payments (WeChat/Alipay)
}

// WebhookEvent represents a parsed webhook event from a payment provider
type WebhookEvent struct {
	Provider          string // "stripe", "alipay", "wechat", "epay"
	EventType         string // "payment.success", "payment.failed", etc.
	TransactionID     string // Provider-specific transaction ID
	SessionID         string // Checkout session ID
	Amount            int64  // Amount in smallest currency unit
	Currency          string
	Status            string // "completed", "failed", "pending"
	CustomerEmail     string
	PaymentMethod     string // "card", "alipay", "wechat_pay", etc.
	// Task 13.5: Renewal fields
	RenewalForLicenseID string // If this is a renewal, the license being renewed
	ExtensionDays       int    // Number of days to extend the license
	Metadata            map[string]string
	ProviderRawEvent    interface{} // Original event object for provider-specific handling
}

// PaymentStatus represents the current status of a payment
type PaymentStatus struct {
	TransactionID string
	Status        string // "pending", "completed", "failed", "refunded"
	Amount        int64
	Currency      string
	PaidAt        *time.Time
	RefundedAt    *time.Time
	Metadata      map[string]string
}

// RefundResult contains the result of a refund operation
type RefundResult struct {
	RefundID      string
	Status        string // "pending", "completed", "failed"
	Amount        int64
	Currency      string
	ProcessedAt   time.Time
}

// PaymentProviderError represents a payment provider specific error
type PaymentProviderError struct {
	Provider string
	Code     string
	Message  string
	Err      error
}

func (e *PaymentProviderError) Error() string {
	if e.Err != nil {
		return e.Provider + ": " + e.Message + " (" + e.Code + "): " + e.Err.Error()
	}
	return e.Provider + ": " + e.Message + " (" + e.Code + ")"
}

func (e *PaymentProviderError) Unwrap() error {
	return e.Err
}
