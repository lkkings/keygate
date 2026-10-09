package wechat

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/tabloy/keygate/internal/payment"
)

// Provider implements the PaymentProvider interface for WeChat Pay
type Provider struct {
	client      *Client
	enabled     bool
	paymentType string // NATIVE, JSAPI, or APP
}

// NewProvider creates a new WeChat Pay provider
func NewProvider(config Config) (*Provider, error) {
	client, err := NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create WeChat Pay client: %w", err)
	}

	// Default to NATIVE mode if not specified
	paymentType := "NATIVE"
	// Note: paymentType configuration would come from settings
	// For now, hardcoded to NATIVE (QR code mode)

	return &Provider{
		client:      client,
		enabled:     true,
		paymentType: paymentType,
	}, nil
}

// Name returns the provider name
func (p *Provider) Name() string {
	return "wechat"
}

// CreateCheckout creates a WeChat Pay checkout (unified order)
func (p *Provider) CreateCheckout(ctx context.Context, params payment.CheckoutParams) (payment.CheckoutResult, error) {
	if !p.enabled {
		return payment.CheckoutResult{}, &payment.ErrProviderDisabled{Provider: "wechat"}
	}

	// Validate amount
	if params.Amount <= 0 {
		return payment.CheckoutResult{}, &payment.ErrInvalidAmount{
			Amount:   params.Amount,
			Currency: params.Currency,
			Reason:   "amount must be positive",
		}
	}

	// Generate order ID from metadata or create one
	outTradeNo := params.Metadata["order_id"]
	if outTradeNo == "" {
		outTradeNo = fmt.Sprintf("WX_%s_%d", params.PlanID, time.Now().Unix())
	}

	// Build unified order request
	req := &UnifiedOrderRequest{
		Body:           fmt.Sprintf("KeyGate License - Plan %s", params.PlanID),
		OutTradeNo:     outTradeNo,
		TotalFee:       int(params.Amount), // Already in cents (分)
		SpbillCreateIP: "127.0.0.1",        // TODO: Get real client IP from context
		NotifyURL:      params.Metadata["notify_url"],
		TradeType:      p.paymentType,
	}

	// For JSAPI mode, openid is required
	if p.paymentType == "JSAPI" {
		openID := params.Metadata["openid"]
		if openID == "" {
			return payment.CheckoutResult{}, &payment.ErrCheckoutFailed{
				Provider: "wechat",
				Reason:   "openid required for JSAPI payment",
			}
		}
		req.OpenID = openID
	}

	// Call unified order API
	resp, err := p.client.UnifiedOrder(ctx, req)
	if err != nil {
		return payment.CheckoutResult{}, &payment.ErrCheckoutFailed{
			Provider: "wechat",
			Reason:   err.Error(),
		}
	}

	result := payment.CheckoutResult{
		SessionID: resp.PrepayID,
	}

	// Set appropriate URL based on trade type
	switch p.paymentType {
	case "NATIVE":
		// QR code URL for scanning
		result.QRCodeURL = resp.CodeURL
		result.CheckoutURL = resp.CodeURL
	case "JSAPI":
		// Return prepay_id for JSAPI SDK
		result.CheckoutURL = "" // Client uses prepay_id directly
	case "APP":
		// Return prepay_id for mobile SDK
		result.CheckoutURL = "" // Client uses prepay_id directly
	}

	return result, nil
}

// VerifyWebhook verifies and parses a WeChat Pay webhook notification
func (p *Provider) VerifyWebhook(ctx context.Context, payload []byte, signature string) (payment.WebhookEvent, error) {
	// Parse XML notification
	params, err := ParseXML(payload)
	if err != nil {
		return payment.WebhookEvent{}, &payment.ErrInvalidSignature{
			Provider: "wechat",
			Reason:   "failed to parse XML: " + err.Error(),
		}
	}

	// Verify signature
	if err := p.client.VerifySign(params); err != nil {
		return payment.WebhookEvent{}, &payment.ErrInvalidSignature{
			Provider: "wechat",
			Reason:   err.Error(),
		}
	}

	// Check return_code
	if params["return_code"] != "SUCCESS" {
		return payment.WebhookEvent{}, fmt.Errorf("webhook return_code is not SUCCESS: %s", params["return_msg"])
	}

	// Build webhook event
	event := payment.WebhookEvent{
		Provider:         "wechat",
		TransactionID:    params["transaction_id"],
		SessionID:        params["out_trade_no"],
		CustomerEmail:    "", // WeChat doesn't provide email in notification
		PaymentMethod:    "wechat_pay",
		Metadata:         extractMetadata(params),
		ProviderRawEvent: params,
	}

	// Parse amount (in cents/分)
	if totalFee := params["total_fee"]; totalFee != "" {
		if fee, err := strconv.ParseInt(totalFee, 10, 64); err == nil {
			event.Amount = fee
		}
	}

	// Currency is always CNY for WeChat Pay
	event.Currency = "CNY"

	// Map result_code to status
	resultCode := params["result_code"]
	if resultCode == "SUCCESS" {
		event.Status = "completed"
		event.EventType = "payment.success"
	} else {
		event.Status = "failed"
		event.EventType = "payment.failed"
	}

	return event, nil
}

// RefundPayment initiates a refund
func (p *Provider) RefundPayment(ctx context.Context, transactionID string, amount int64, reason string) (payment.RefundResult, error) {
	if !p.enabled {
		return payment.RefundResult{}, &payment.ErrProviderDisabled{Provider: "wechat"}
	}

	// Generate unique refund ID
	outRefundNo := fmt.Sprintf("REFUND_%s_%d", transactionID, time.Now().Unix())

	// Build refund request
	// Note: We need the original total_fee, which should be looked up from transaction records
	// For now, using the refund amount as total (assuming full refund)
	req := &RefundRequest{
		OutTradeNo:  transactionID,
		OutRefundNo: outRefundNo,
		TotalFee:    int(amount), // TODO: Should be original order amount
		RefundFee:   int(amount),
		RefundDesc:  reason,
	}

	// Call refund API
	resp, err := p.client.RefundOrder(ctx, req)
	if err != nil {
		return payment.RefundResult{}, &payment.ErrRefundFailed{
			TransactionID: transactionID,
			Provider:      "wechat",
			Reason:        err.Error(),
		}
	}

	return payment.RefundResult{
		RefundID:    resp.RefundID,
		Status:      "completed",
		Amount:      int64(resp.RefundFee),
		Currency:    "CNY",
		ProcessedAt: time.Now(),
	}, nil
}

// GetPaymentStatus retrieves the current status of a payment
func (p *Provider) GetPaymentStatus(ctx context.Context, transactionID string) (payment.PaymentStatus, error) {
	// Query order
	resp, err := p.client.QueryOrder(ctx, transactionID)
	if err != nil {
		return payment.PaymentStatus{}, &payment.ErrPaymentNotFound{
			TransactionID: transactionID,
			Provider:      "wechat",
		}
	}

	status := payment.PaymentStatus{
		TransactionID: resp.TransactionID,
		Amount:        int64(resp.TotalFee),
		Currency:      "CNY",
		Metadata: map[string]string{
			"out_trade_no": resp.OutTradeNo,
			"trade_state":  resp.TradeState,
		},
	}

	// Map trade_state to our status
	switch resp.TradeState {
	case "SUCCESS":
		status.Status = "completed"
		if resp.TimeEnd != "" {
			// Parse time_end (format: yyyyMMddHHmmss)
			if t, err := time.Parse("20060102150405", resp.TimeEnd); err == nil {
				status.PaidAt = &t
			}
		}
	case "REFUND":
		status.Status = "refunded"
	case "NOTPAY":
		status.Status = "pending"
	case "CLOSED":
		status.Status = "failed"
	case "REVOKED":
		status.Status = "cancelled"
	case "USERPAYING":
		status.Status = "pending"
	case "PAYERROR":
		status.Status = "failed"
	default:
		status.Status = resp.TradeState
	}

	return status, nil
}

// extractMetadata extracts custom metadata from WeChat parameters
func extractMetadata(params map[string]string) map[string]string {
	metadata := make(map[string]string)

	// Store original out_trade_no for reference
	if outTradeNo := params["out_trade_no"]; outTradeNo != "" {
		metadata["out_trade_no"] = outTradeNo
	}

	// Store additional fields that might be useful
	if openID := params["openid"]; openID != "" {
		metadata["openid"] = openID
	}

	if tradeType := params["trade_type"]; tradeType != "" {
		metadata["trade_type"] = tradeType
	}

	if bankType := params["bank_type"]; bankType != "" {
		metadata["bank_type"] = bankType
	}

	if timeEnd := params["time_end"]; timeEnd != "" {
		metadata["time_end"] = timeEnd
	}

	return metadata
}
