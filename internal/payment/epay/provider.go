package epay

import (
	"context"
	"fmt"

	"github.com/tabloy/keygate/internal/payment"
)

// Provider implements the PaymentProvider interface for ePay
// NOTE: This is a stub implementation pending official ePay API documentation
type Provider struct {
	client            *Client
	enabled           bool
	signatureAlgorithm SignatureAlgorithm
}

// Config holds ePay provider configuration
type ProviderConfig struct {
	MerchantID        string
	APIKey            string
	Gateway           string
	IsSandbox         bool
	SignatureAlgorithm string // "MD5" or "HMAC-SHA256"
}

// NewProvider creates a new ePay provider
func NewProvider(config ProviderConfig) (*Provider, error) {
	clientConfig := Config{
		MerchantID: config.MerchantID,
		APIKey:     config.APIKey,
		Gateway:    config.Gateway,
		IsSandbox:  config.IsSandbox,
	}

	client, err := NewClient(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create ePay client: %w", err)
	}

	// Default to MD5 if not specified
	sigAlgorithm := SignatureMD5
	if config.SignatureAlgorithm == "HMAC-SHA256" {
		sigAlgorithm = SignatureHMACSHA256
	}

	return &Provider{
		client:            client,
		enabled:           true,
		signatureAlgorithm: sigAlgorithm,
	}, nil
}

// Name returns the provider name
func (p *Provider) Name() string {
	return "epay"
}

// CreateCheckout creates an ePay checkout session
// TODO: Implement once ePay API is documented
func (p *Provider) CreateCheckout(ctx context.Context, params payment.CheckoutParams) (payment.CheckoutResult, error) {
	if !p.enabled {
		return payment.CheckoutResult{}, &payment.ErrProviderDisabled{Provider: "epay"}
	}

	// TODO: Implement actual ePay checkout creation
	// For now, return not implemented error
	return payment.CheckoutResult{}, &payment.ErrNotImplemented{
		Provider: "epay",
		Feature:  "CreateCheckout - awaiting official API documentation",
	}
}

// VerifyWebhook verifies and parses an ePay webhook notification
// TODO: Implement once ePay webhook format is documented
func (p *Provider) VerifyWebhook(ctx context.Context, payload []byte, signature string) (payment.WebhookEvent, error) {
	// TODO: Parse payload format (JSON? XML? Form-encoded?)
	// TODO: Extract parameters and verify signature
	// TODO: Map ePay status codes to our webhook event format

	return payment.WebhookEvent{}, &payment.ErrNotImplemented{
		Provider: "epay",
		Feature:  "VerifyWebhook - awaiting webhook format specification",
	}
}

// RefundPayment initiates a refund
// TODO: Implement once ePay refund API is documented
func (p *Provider) RefundPayment(ctx context.Context, transactionID string, amount int64, reason string) (payment.RefundResult, error) {
	if !p.enabled {
		return payment.RefundResult{}, &payment.ErrProviderDisabled{Provider: "epay"}
	}

	// TODO: Implement actual ePay refund
	return payment.RefundResult{}, &payment.ErrNotImplemented{
		Provider: "epay",
		Feature:  "RefundPayment - awaiting official API documentation",
	}
}

// GetPaymentStatus retrieves the current status of a payment
// TODO: Implement once ePay query API is documented
func (p *Provider) GetPaymentStatus(ctx context.Context, transactionID string) (payment.PaymentStatus, error) {
	// TODO: Implement actual ePay status query
	return payment.PaymentStatus{}, &payment.ErrNotImplemented{
		Provider: "epay",
		Feature:  "GetPaymentStatus - awaiting official API documentation",
	}
}
