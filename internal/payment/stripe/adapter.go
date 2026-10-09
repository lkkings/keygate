package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/tabloy/keygate/internal/payment"
	"github.com/tabloy/keygate/internal/store"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/paymentintent"
	"github.com/stripe/stripe-go/v82/refund"
	"github.com/stripe/stripe-go/v82/webhook"
)

// Provider is a PaymentProvider implementation for Stripe
type Provider struct {
	apiKey        string
	webhookSecret string
	store         *store.Store
}

// Config holds Stripe provider configuration
type Config struct {
	APIKey        string
	WebhookSecret string
}

// NewProvider creates a new Stripe payment provider
func NewProvider(config Config, store *store.Store) *Provider {
	// Set Stripe API key
	stripe.Key = config.APIKey

	return &Provider{
		apiKey:        config.APIKey,
		webhookSecret: config.WebhookSecret,
		store:         store,
	}
}

// Name returns the provider name
func (p *Provider) Name() string {
	return "stripe"
}

// CreateCheckout creates a Stripe checkout session
func (p *Provider) CreateCheckout(ctx context.Context, params payment.CheckoutParams) (payment.CheckoutResult, error) {
	// Build line items from amount and currency
	var lineItems []*stripe.CheckoutSessionLineItemParams

	if priceID, ok := params.Metadata["price_id"]; ok {
		// Use existing Stripe price ID if provided
		lineItems = append(lineItems, &stripe.CheckoutSessionLineItemParams{
			Price:    stripe.String(priceID),
			Quantity: stripe.Int64(1),
		})
	} else {
		// Create price data dynamically
		lineItems = append(lineItems, &stripe.CheckoutSessionLineItemParams{
			PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
				Currency: stripe.String(params.Currency),
				ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
					Name:        stripe.String(params.ProductID),
					Description: stripe.String(fmt.Sprintf("License for %s - %s", params.ProductID, params.PlanID)),
				},
				UnitAmount: stripe.Int64(params.Amount),
			},
			Quantity: stripe.Int64(1),
		})
	}

	// Prepare Stripe checkout session params
	sessionParams := &stripe.CheckoutSessionParams{
		Mode:          stripe.String("payment"), // One-time payment
		SuccessURL:    stripe.String(params.SuccessURL),
		CancelURL:     stripe.String(params.CancelURL),
		CustomerEmail: stripe.String(params.CustomerEmail),
		LineItems:     lineItems,
		Metadata:      params.Metadata,
	}

	// Create the checkout session
	sess, err := session.New(sessionParams)
	if err != nil {
		return payment.CheckoutResult{}, &payment.ErrCheckoutFailed{
			Provider: "stripe",
			Reason:   err.Error(),
		}
	}

	return payment.CheckoutResult{
		CheckoutURL: sess.URL,
		SessionID:   sess.ID,
		QRCodeURL:   "", // Stripe doesn't use QR codes
	}, nil
}

// VerifyWebhook verifies a Stripe webhook signature
func (p *Provider) VerifyWebhook(ctx context.Context, payload []byte, signature string) (payment.WebhookEvent, error) {
	if p.webhookSecret == "" {
		return payment.WebhookEvent{}, &payment.ErrInvalidSignature{
			Provider: "stripe",
			Reason:   "webhook secret not configured",
		}
	}

	// Verify the webhook signature
	event, err := webhook.ConstructEvent(payload, signature, p.webhookSecret)
	if err != nil {
		return payment.WebhookEvent{}, &payment.ErrInvalidSignature{
			Provider: "stripe",
			Reason:   err.Error(),
		}
	}

	// Parse the event into our common webhook event format
	webhookEvent := payment.WebhookEvent{
		Provider:  "stripe",
		EventType: string(event.Type),
		Metadata:  make(map[string]string),
	}

	// Extract common fields based on event type
	switch event.Type {
	case "checkout.session.completed":
		var sess stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &sess); err == nil {
			webhookEvent.SessionID = sess.ID
			webhookEvent.TransactionID = sess.PaymentIntent.ID
			webhookEvent.Amount = sess.AmountTotal
			webhookEvent.Currency = string(sess.Currency)
			webhookEvent.Status = "completed"
			webhookEvent.CustomerEmail = sess.CustomerDetails.Email
			webhookEvent.PaymentMethod = "card"
			for k, v := range sess.Metadata {
				webhookEvent.Metadata[k] = v
			}
		}
	case "payment_intent.succeeded":
		var pi stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &pi); err == nil {
			webhookEvent.TransactionID = pi.ID
			webhookEvent.Amount = pi.Amount
			webhookEvent.Currency = string(pi.Currency)
			webhookEvent.Status = "completed"
			for k, v := range pi.Metadata {
				webhookEvent.Metadata[k] = v
			}
		}
	case "payment_intent.payment_failed":
		var pi stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &pi); err == nil {
			webhookEvent.TransactionID = pi.ID
			webhookEvent.Amount = pi.Amount
			webhookEvent.Currency = string(pi.Currency)
			webhookEvent.Status = "failed"
			for k, v := range pi.Metadata {
				webhookEvent.Metadata[k] = v
			}
		}
	case "charge.refunded":
		var charge stripe.Charge
		if err := json.Unmarshal(event.Data.Raw, &charge); err == nil {
			webhookEvent.TransactionID = charge.PaymentIntent.ID
			webhookEvent.Amount = charge.AmountRefunded
			webhookEvent.Currency = string(charge.Currency)
			webhookEvent.Status = "refunded"
			for k, v := range charge.Metadata {
				webhookEvent.Metadata[k] = v
			}
		}
	}

	return webhookEvent, nil
}

// FulfillPayment fulfills a payment after successful checkout
// RefundPayment processes a refund through Stripe
func (p *Provider) RefundPayment(ctx context.Context, transactionID string, amount int64, reason string) (payment.RefundResult, error) {
	params := &stripe.RefundParams{
		PaymentIntent: stripe.String(transactionID),
	}

	if amount > 0 {
		params.Amount = stripe.Int64(amount)
	}

	if reason != "" {
		params.Reason = stripe.String(reason)
	}

	r, err := refund.New(params)
	if err != nil {
		return payment.RefundResult{}, &payment.ErrRefundFailed{
			TransactionID: transactionID,
			Provider:      "stripe",
			Reason:        err.Error(),
		}
	}

	return payment.RefundResult{
		RefundID:    r.ID,
		Status:      string(r.Status),
		Amount:      r.Amount,
		Currency:    string(r.Currency),
		ProcessedAt: time.Unix(r.Created, 0),
	}, nil
}

// GetPaymentStatus retrieves the status of a payment intent
func (p *Provider) GetPaymentStatus(ctx context.Context, transactionID string) (payment.PaymentStatus, error) {
	pi, err := paymentintent.Get(transactionID, nil)
	if err != nil {
		return payment.PaymentStatus{}, &payment.ErrPaymentNotFound{
			TransactionID: transactionID,
			Provider:      "stripe",
		}
	}

	status := payment.PaymentStatus{
		TransactionID: pi.ID,
		Status:        string(pi.Status),
		Amount:        pi.Amount,
		Currency:      string(pi.Currency),
		Metadata:      pi.Metadata,
	}

	if pi.Created > 0 {
		createdAt := time.Unix(pi.Created, 0)
		status.PaidAt = &createdAt
	}

	return status, nil
}
