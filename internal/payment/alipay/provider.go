package alipay

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/tabloy/keygate/internal/payment"
)

// Provider implements the PaymentProvider interface for Alipay
type Provider struct {
	client  *Client
	enabled bool
	appID   string
}

// NewProvider creates a new Alipay payment provider
func NewProvider(config Config) (*Provider, error) {
	client, err := NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Alipay client: %w", err)
	}

	return &Provider{
		client:  client,
		enabled: true,
		appID:   config.AppID,
	}, nil
}

// Name returns the provider name
func (p *Provider) Name() string {
	return "alipay"
}

// CreateCheckout creates an Alipay payment checkout
func (p *Provider) CreateCheckout(ctx context.Context, params payment.CheckoutParams) (payment.CheckoutResult, error) {
	if !p.enabled {
		return payment.CheckoutResult{}, &payment.ErrProviderDisabled{Provider: "alipay"}
	}

	// Generate a unique order ID
	outTradeNo := params.Metadata["order_id"]
	if outTradeNo == "" {
		outTradeNo = fmt.Sprintf("ORDER_%s_%d", params.PlanID, time.Now().Unix())
	}

	// Convert amount from cents to yuan (Alipay uses yuan with 2 decimal places)
	totalAmount := float64(params.Amount) / 100.0

	// Build Alipay order parameters
	orderParams := map[string]interface{}{
		"out_trade_no": outTradeNo,
		"product_code": "FAST_INSTANT_TRADE_PAY", // Standard product code for web payments
		"total_amount": fmt.Sprintf("%.2f", totalAmount),
		"subject":      fmt.Sprintf("KeyGate License - Plan %s", params.PlanID),
		"body":         params.Metadata["description"],
		"return_url":   params.SuccessURL,
		"notify_url":   params.Metadata["notify_url"], // Async notification URL
	}

	// Add optional fields from metadata
	if params.ProductID != "" {
		orderParams["goods_type"] = "1" // Virtual goods
	}

	// Create the order
	resp, err := p.client.UnifiedOrder(ctx, orderParams)
	if err != nil {
		return payment.CheckoutResult{}, &payment.ErrCheckoutFailed{
			Provider: "alipay",
			Reason:   err.Error(),
		}
	}

	return payment.CheckoutResult{
		CheckoutURL: resp.RedirectURL,
		SessionID:   resp.OutTradeNo,
		QRCodeURL:   "", // Can be generated from RedirectURL if needed for mobile
	}, nil
}

// VerifyWebhook verifies an Alipay webhook notification
func (p *Provider) VerifyWebhook(ctx context.Context, payload []byte, signature string) (payment.WebhookEvent, error) {
	// Parse form parameters from payload
	params, err := DecodeParams(string(payload))
	if err != nil {
		return payment.WebhookEvent{}, &payment.ErrInvalidSignature{
			Provider: "alipay",
			Reason:   "failed to parse parameters: " + err.Error(),
		}
	}

	// Verify signature
	if err := VerifyNotifySign(params, p.client.AlipayPublicKey); err != nil {
		return payment.WebhookEvent{}, &payment.ErrInvalidSignature{
			Provider: "alipay",
			Reason:   err.Error(),
		}
	}

	// Parse webhook event
	event := payment.WebhookEvent{
		Provider:         "alipay",
		EventType:        "payment." + params["trade_status"],
		TransactionID:    params["trade_no"],
		SessionID:        params["out_trade_no"],
		CustomerEmail:    params["buyer_email"],
		PaymentMethod:    "alipay",
		Metadata:         extractMetadata(params),
		ProviderRawEvent: params,
	}

	// Parse amount (in yuan)
	if totalAmount, err := strconv.ParseFloat(params["total_amount"], 64); err == nil {
		event.Amount = int64(totalAmount * 100) // Convert to cents
	}

	// Set currency (Alipay typically uses CNY)
	event.Currency = "CNY"

	// Map trade status to our status
	tradeStatus := params["trade_status"]
	switch tradeStatus {
	case "TRADE_SUCCESS", "TRADE_FINISHED":
		event.Status = "completed"
	case "TRADE_CLOSED":
		event.Status = "cancelled"
	case "WAIT_BUYER_PAY":
		event.Status = "pending"
	default:
		event.Status = "unknown"
	}

	return event, nil
}

// FulfillPayment processes a successful payment
// RefundPayment initiates a refund
func (p *Provider) RefundPayment(ctx context.Context, transactionID string, amount int64, reason string) (payment.RefundResult, error) {
	if !p.enabled {
		return payment.RefundResult{}, &payment.ErrProviderDisabled{Provider: "alipay"}
	}

	// Convert amount from cents to yuan
	refundAmount := float64(amount) / 100.0

	// Call Alipay refund API
	resp, err := p.client.RefundOrder(ctx, transactionID, refundAmount, reason)
	if err != nil {
		return payment.RefundResult{}, &payment.ErrRefundFailed{
			TransactionID: transactionID,
			Provider:      "alipay",
			Reason:        err.Error(),
		}
	}

	// Check if refund was successful
	if resp.Code != "10000" {
		return payment.RefundResult{}, &payment.ErrRefundFailed{
			TransactionID: transactionID,
			Provider:      "alipay",
			Reason:        fmt.Sprintf("refund failed with code %s: %s", resp.Code, resp.Msg),
		}
	}

	// Parse refund amount
	refundFee, _ := strconv.ParseFloat(resp.RefundFee, 64)

	// Parse refund time
	refundTime := time.Now()
	if resp.GMTRefundPay != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", resp.GMTRefundPay); err == nil {
			refundTime = t
		}
	}

	return payment.RefundResult{
		RefundID:    resp.TradeNo,
		Status:      "completed",
		Amount:      int64(refundFee * 100), // Convert to cents
		Currency:    "CNY",
		ProcessedAt: refundTime,
	}, nil
}

// GetPaymentStatus retrieves the status of a payment
func (p *Provider) GetPaymentStatus(ctx context.Context, transactionID string) (payment.PaymentStatus, error) {
	resp, err := p.client.QueryOrder(ctx, transactionID)
	if err != nil {
		return payment.PaymentStatus{}, &payment.ErrPaymentNotFound{
			TransactionID: transactionID,
			Provider:      "alipay",
		}
	}

	// Check response code
	if resp.Code != "10000" {
		return payment.PaymentStatus{}, fmt.Errorf("query failed with code %s: %s", resp.Code, resp.Msg)
	}

	// Parse amount
	totalAmount, _ := strconv.ParseFloat(resp.TotalAmount, 64)

	status := payment.PaymentStatus{
		TransactionID: resp.TradeNo,
		Amount:        int64(totalAmount * 100), // Convert to cents
		Currency:      "CNY",
		Metadata:      map[string]string{
			"out_trade_no":    resp.OutTradeNo,
			"buyer_logon_id":  resp.BuyerLogonID,
		},
	}

	// Map trade status
	switch resp.TradeStatus {
	case "TRADE_SUCCESS", "TRADE_FINISHED":
		status.Status = "completed"
		// Note: Alipay doesn't provide exact payment time in query response
		// Would need to store it from the notification
	case "TRADE_CLOSED":
		status.Status = "failed"
	case "WAIT_BUYER_PAY":
		status.Status = "pending"
	default:
		status.Status = resp.TradeStatus
	}

	return status, nil
}

// extractMetadata extracts custom metadata from Alipay parameters
func extractMetadata(params map[string]string) map[string]string {
	metadata := make(map[string]string)

	// Extract known fields
	if passbackParams := params["passback_params"]; passbackParams != "" {
		// passback_params can contain custom data
		metadata["passback_params"] = passbackParams
	}

	// Store original out_trade_no for reference
	if outTradeNo := params["out_trade_no"]; outTradeNo != "" {
		metadata["out_trade_no"] = outTradeNo
	}

	return metadata
}
