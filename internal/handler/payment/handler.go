package payment

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tabloy/keygate/internal/payment"
	"github.com/tabloy/keygate/internal/service"
)

// Handler handles payment-related HTTP requests
type Handler struct {
	PaymentService *service.PaymentService
	Logger         *slog.Logger
}

// NewHandler creates a new payment handler
func NewHandler(paymentService *service.PaymentService, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return &Handler{
		PaymentService: paymentService,
		Logger:         logger,
	}
}

// CreateCheckoutRequest represents a checkout creation request
type CreateCheckoutRequest struct {
	PlanID      string            `json:"plan_id" binding:"required"`
	ProductID   string            `json:"product_id"`
	CustomerEmail string          `json:"customer_email" binding:"required,email"`
	SuccessURL  string            `json:"success_url" binding:"required,url"`
	CancelURL   string            `json:"cancel_url" binding:"required,url"`
	Metadata    map[string]string `json:"metadata"`
}

// CreateCheckoutResponse represents a checkout creation response
type CreateCheckoutResponse struct {
	CheckoutURL string `json:"checkout_url"`
	SessionID   string `json:"session_id"`
	QRCodeURL   string `json:"qr_code_url,omitempty"`
}

// CreateCheckout creates a payment checkout session
// POST /api/v1/payment/:provider/checkout
func (h *Handler) CreateCheckout(c *gin.Context) {
	providerName := c.Param("provider")

	var req CreateCheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build checkout parameters
	params := payment.CheckoutParams{
		PlanID:        req.PlanID,
		ProductID:     req.ProductID,
		Amount:        0, // Will be set based on plan price
		Currency:      "USD", // Will be set based on provider and plan
		CustomerEmail: req.CustomerEmail,
		SuccessURL:    req.SuccessURL,
		CancelURL:     req.CancelURL,
		Metadata:      req.Metadata,
	}

	// TODO: Fetch plan details and set amount/currency based on provider
	// For now, using placeholder values

	result, err := h.PaymentService.CreateCheckout(c.Request.Context(), providerName, params)
	if err != nil {
		h.Logger.Error("failed to create checkout",
			"provider", providerName,
			"error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create checkout"})
		return
	}

	c.JSON(http.StatusOK, CreateCheckoutResponse{
		CheckoutURL: result.CheckoutURL,
		SessionID:   result.SessionID,
		QRCodeURL:   result.QRCodeURL,
	})
}

// HandleWebhook processes a webhook from a payment provider
// POST /api/v1/payment/:provider/webhook
func (h *Handler) HandleWebhook(c *gin.Context) {
	providerName := c.Param("provider")

	// Read the raw body
	body, err := c.GetRawData()
	if err != nil {
		h.Logger.Error("failed to read webhook body",
			"provider", providerName,
			"error", err)
		c.Status(http.StatusBadRequest)
		return
	}

	// Get signature from header (varies by provider)
	var signature string
	switch providerName {
	case "stripe":
		signature = c.GetHeader("Stripe-Signature")
	case "alipay":
		// Alipay sends signature in the body, not header
		signature = ""
	case "wechat":
		// WeChat doesn't use header signatures
		signature = ""
	default:
		signature = c.GetHeader("X-Signature")
	}

	// Process the webhook
	if err := h.PaymentService.HandleWebhook(c.Request.Context(), providerName, body, signature); err != nil {
		h.Logger.Error("webhook processing failed",
			"provider", providerName,
			"error", err)
		c.Status(http.StatusBadRequest)
		return
	}

	// Return success response (format varies by provider)
	switch providerName {
	case "stripe":
		c.Status(http.StatusOK)
	case "alipay":
		c.String(http.StatusOK, "success")
	case "wechat":
		// WeChat requires XML response
		c.XML(http.StatusOK, gin.H{
			"return_code": "SUCCESS",
			"return_msg":  "OK",
		})
	default:
		c.Status(http.StatusOK)
	}
}

// HandleReturn handles synchronous payment returns (e.g., Alipay redirect)
// GET /api/v1/payment/:provider/return
func (h *Handler) HandleReturn(c *gin.Context) {
	providerName := c.Param("provider")

	// For Alipay, verify the return parameters
	queryParams := c.Request.URL.Query()

	// Convert query parameters to map
	params := make(map[string]string)
	for key, values := range queryParams {
		if len(values) > 0 {
			params[key] = values[0]
		}
	}

	// Verify signature
	paramsJSON, _ := json.Marshal(params)
	signature := params["sign"]

	err := h.PaymentService.HandleWebhook(c.Request.Context(), providerName, paramsJSON, signature)
	if err != nil {
		h.Logger.Error("return verification failed",
			"provider", providerName,
			"error", err)
		// Redirect to failure page
		c.Redirect(http.StatusFound, "/payment/failed")
		return
	}

	// Redirect to success page with transaction info
	transactionID := params["out_trade_no"]
	successURL := fmt.Sprintf("/payment/success?transaction_id=%s", transactionID)
	c.Redirect(http.StatusFound, successURL)
}

// GetProviders returns a list of enabled payment providers
// GET /api/v1/payment/providers
func (h *Handler) GetProviders(c *gin.Context) {
	providerNames := payment.ListProviders()

	response := make([]map[string]interface{}, 0, len(providerNames))
	for _, name := range providerNames {
		response = append(response, map[string]interface{}{
			"name": name,
			// Add more provider metadata as needed
		})
	}

	c.JSON(http.StatusOK, gin.H{"providers": response})
}

// RefundRequest represents a refund request
type RefundRequest struct {
	Amount int64  `json:"amount" binding:"required"`
	Reason string `json:"reason" binding:"required"`
}

// RefundPayment initiates a refund for a payment
// POST /api/v1/payment/:provider/refund/:transaction_id
// Requires admin authentication
func (h *Handler) RefundPayment(c *gin.Context) {
	providerName := c.Param("provider")
	transactionID := c.Param("transaction_id")

	var req RefundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: Verify admin authentication

	result, err := h.PaymentService.RefundPayment(c.Request.Context(), transactionID, req.Amount, req.Reason)
	if err != nil {
		h.Logger.Error("refund failed",
			"provider", providerName,
			"transaction_id", transactionID,
			"error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "refund failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"refund_id": result.RefundID,
		"status":    result.Status,
		"amount":    result.Amount,
		"currency":  result.Currency,
	})
}

// GetPaymentStatus retrieves the status of a payment
// GET /api/v1/payment/:provider/status/:transaction_id
func (h *Handler) GetPaymentStatus(c *gin.Context) {
	providerName := c.Param("provider")
	transactionID := c.Param("transaction_id")

	status, err := h.PaymentService.GetPaymentStatus(c.Request.Context(), providerName, transactionID)
	if err != nil {
		h.Logger.Error("failed to get payment status",
			"provider", providerName,
			"transaction_id", transactionID,
			"error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "transaction not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transaction_id": status.TransactionID,
		"status":         status.Status,
		"amount":         status.Amount,
		"currency":       status.Currency,
		"paid_at":        status.PaidAt,
		"metadata":       status.Metadata,
	})
}
