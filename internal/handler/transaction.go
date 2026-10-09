package handler

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tabloy/keygate/internal/model"
	"github.com/tabloy/keygate/internal/store"
)

// TransactionHandler handles transaction history and reporting
type TransactionHandler struct {
	Store *store.Store
}

// NewTransactionHandler creates a new transaction handler
func NewTransactionHandler(store *store.Store) *TransactionHandler {
	return &TransactionHandler{
		Store: store,
	}
}

// ListTransactions handles GET /api/v1/admin/transactions
// Task 17.1: Return paginated transactions
// Task 17.2: Support filters for provider, status, date range
func (h *TransactionHandler) ListTransactions(c *gin.Context) {
	// Pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "50"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 50
	}
	offset := (page - 1) * perPage

	// Task 17.2: Filters
	provider := c.Query("provider")
	status := c.Query("status")
	dateFrom := c.Query("date_from")
	dateTo := c.Query("date_to")

	// Build query
	query := h.Store.DB.NewSelect().Model((*model.PaymentTransaction)(nil))

	if provider != "" {
		query = query.Where("provider_name = ?", provider)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if dateFrom != "" {
		query = query.Where("created_at >= ?", dateFrom)
	}
	if dateTo != "" {
		query = query.Where("created_at <= ?", dateTo)
	}

	// Get total count
	total, err := query.Count(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count transactions"})
		return
	}

	// Get transactions
	var transactions []*model.PaymentTransaction
	err = query.
		Order("created_at DESC").
		Limit(perPage).
		Offset(offset).
		Scan(c.Request.Context(), &transactions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch transactions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transactions": transactions,
		"pagination": gin.H{
			"page":     page,
			"per_page": perPage,
			"total":    total,
		},
	})
}

// GetRevenueReport handles GET /api/v1/admin/transactions/revenue
// Task 17.4: Revenue report grouped by currency
func (h *TransactionHandler) GetRevenueReport(c *gin.Context) {
	dateFrom := c.Query("date_from")
	dateTo := c.Query("date_to")

	query := h.Store.DB.NewSelect().
		Model((*model.PaymentTransaction)(nil)).
		Column("currency").
		ColumnExpr("SUM(amount) as total_amount").
		ColumnExpr("COUNT(*) as transaction_count").
		Where("status = ?", "completed").
		Group("currency")

	if dateFrom != "" {
		query = query.Where("created_at >= ?", dateFrom)
	}
	if dateTo != "" {
		query = query.Where("created_at <= ?", dateTo)
	}

	var results []struct {
		Currency         string `bun:"currency"`
		TotalAmount      int64  `bun:"total_amount"`
		TransactionCount int    `bun:"transaction_count"`
	}

	err := query.Scan(c.Request.Context(), &results)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate revenue report"})
		return
	}

	// Task 17.4: Format report by currency
	report := make(map[string]gin.H)
	for _, r := range results {
		report[r.Currency] = gin.H{
			"total_cents":       r.TotalAmount,
			"total_formatted":   fmt.Sprintf("%.2f", float64(r.TotalAmount)/100),
			"transaction_count": r.TransactionCount,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"revenue_by_currency": report,
		"date_from":           dateFrom,
		"date_to":             dateTo,
	})
}

// ExportTransactionsCSV handles GET /api/v1/admin/transactions/export
// Task 17.5: CSV export for transactions
func (h *TransactionHandler) ExportTransactionsCSV(c *gin.Context) {
	// Apply same filters as list
	provider := c.Query("provider")
	status := c.Query("status")
	dateFrom := c.Query("date_from")
	dateTo := c.Query("date_to")

	query := h.Store.DB.NewSelect().Model((*model.PaymentTransaction)(nil))

	if provider != "" {
		query = query.Where("provider_name = ?", provider)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if dateFrom != "" {
		query = query.Where("created_at >= ?", dateFrom)
	}
	if dateTo != "" {
		query = query.Where("created_at <= ?", dateTo)
	}

	var transactions []*model.PaymentTransaction
	err := query.
		Order("created_at DESC").
		Limit(10000). // Reasonable limit for CSV export
		Scan(c.Request.Context(), &transactions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch transactions"})
		return
	}

	// Set CSV headers
	filename := fmt.Sprintf("transactions_%s.csv", time.Now().Format("2006-01-02"))
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	// Write CSV
	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	// Header row
	writer.Write([]string{
		"ID",
		"Provider",
		"Transaction ID",
		"Amount",
		"Currency",
		"Status",
		"Customer Email",
		"Refund Amount",
		"Created At",
		"Refunded At",
	})

	// Data rows
	for _, txn := range transactions {
		refundedAt := ""
		if txn.RefundedAt != nil {
			refundedAt = txn.RefundedAt.Format(time.RFC3339)
		}

		writer.Write([]string{
			fmt.Sprintf("%d", txn.ID),
			txn.ProviderName,
			txn.ProviderTxID,
			fmt.Sprintf("%.2f", float64(txn.Amount)/100),
			txn.Currency,
			txn.Status,
			txn.CustomerEmail,
			fmt.Sprintf("%.2f", float64(txn.RefundAmount)/100),
			txn.CreatedAt.Format(time.RFC3339),
			refundedAt,
		})
	}
}
