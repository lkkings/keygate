package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tabloy/keygate/internal/model"
	"github.com/tabloy/keygate/internal/service"
	"github.com/tabloy/keygate/internal/store"
)

// RefundHandler handles payment refund operations
type RefundHandler struct {
	Store      *store.Store
	PaymentSvc *service.PaymentService
	EmailSvc   *service.EmailService
}

// NewRefundHandler creates a new refund handler
func NewRefundHandler(store *store.Store, paymentSvc *service.PaymentService, emailSvc *service.EmailService) *RefundHandler {
	return &RefundHandler{
		Store:      store,
		PaymentSvc: paymentSvc,
		EmailSvc:   emailSvc,
	}
}

// ProcessRefund handles POST /api/v1/payment/{provider}/refund
// Task 16.3: Process refund request
func (h *RefundHandler) ProcessRefund(c *gin.Context) {
	provider := c.Param("provider")

	var req struct {
		TransactionID string  `json:"transaction_id" binding:"required"`
		Amount        int64   `json:"amount"` // 0 or omitted = full refund
		Reason        string  `json:"reason"`
		RevokeLicense bool    `json:"revoke_license"` // Task 16.5: Immediate revocation flag
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	ctx := c.Request.Context()

	// Load transaction
	var txn model.PaymentTransaction
	err := h.Store.DB.NewSelect().
		Model(&txn).
		Where("provider_tx_id = ? AND provider_name = ?", req.TransactionID, provider).
		Scan(ctx)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "transaction not found"})
		return
	}

	// Check if already refunded
	if txn.Status == "refunded" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "transaction already refunded"})
		return
	}

	// Default to full refund
	refundAmount := req.Amount
	if refundAmount == 0 {
		refundAmount = txn.Amount
	}

	// Validate refund amount
	if refundAmount > txn.Amount {
		c.JSON(http.StatusBadRequest, gin.H{"error": "refund amount exceeds transaction amount"})
		return
	}

	// Call payment provider to process refund
	result, err := h.PaymentSvc.RefundPayment(ctx, req.TransactionID, refundAmount, req.Reason)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "refund failed", "details": err.Error()})
		return
	}

	// Task 16.4: Update transaction record
	now := time.Now()
	_, err = h.Store.DB.NewUpdate().
		Model(&txn).
		Set("status = ?", "refunded").
		Set("refund_amount = ?", refundAmount).
		Set("refunded_at = ?", now).
		Set("updated_at = ?", now).
		Where("id = ?", txn.ID).
		Exec(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update transaction"})
		return
	}

	// Task 16.5: Update license status if requested
	if req.RevokeLicense && txn.LicenseID != "" {
		var license model.License
		err = h.Store.DB.NewSelect().Model(&license).Where("id = ?", txn.LicenseID).Scan(ctx)
		if err == nil {
			_, err = h.Store.DB.NewUpdate().
				Model(&license).
				Set("status = ?", model.StatusRevoked).
				Set("updated_at = ?", now).
				Where("id = ?", license.ID).
				Exec(ctx)
			if err != nil {
				// Log error but don't fail the refund
				c.Set("license_revoke_error", err.Error())
			}
		}
	}

	// Task 16.6: Send refund confirmation email
	if txn.CustomerEmail != "" {
		subject := "Refund Processed"
		body := "Your payment has been refunded. The refund amount will appear in your account within 5-10 business days."

		err = h.EmailSvc.Send(txn.CustomerEmail, subject, body)
		if err != nil {
			// Log error but don't fail the refund
			c.Set("email_error", err.Error())
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"refund_id":     result.RefundID,
		"refund_amount": refundAmount,
		"status":        result.Status,
	})
}

// GetTransactionForRefund loads transaction details for refund modal
// Task 16.2: Provide transaction details for confirmation
func (h *RefundHandler) GetTransactionForRefund(c *gin.Context) {
	transactionID := c.Param("transaction_id")
	provider := c.Query("provider")

	var txn model.PaymentTransaction
	err := h.Store.DB.NewSelect().
		Model(&txn).
		Where("provider_tx_id = ? AND provider_name = ?", transactionID, provider).
		Scan(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "transaction not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transaction_id": txn.ProviderTxID,
		"amount":         txn.Amount,
		"currency":       txn.Currency,
		"status":         txn.Status,
		"customer_email": txn.CustomerEmail,
		"created_at":     txn.CreatedAt,
		"can_refund":     txn.Status == "completed",
	})
}
