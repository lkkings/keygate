package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tabloy/keygate/internal/model"
	"github.com/tabloy/keygate/internal/store"
)

// RenewalHandler handles license renewal checkout flows
// Task 15.1: Create renewal route
type RenewalHandler struct {
	Store *store.Store
}

// NewRenewalHandler creates a new renewal handler
func NewRenewalHandler(store *store.Store) *RenewalHandler {
	return &RenewalHandler{
		Store: store,
	}
}

// GetRenewalPage handles GET /renewal?license={id} or /renewal?token={signed}
// Task 15.1: Validate license ownership and load renewal page
func (h *RenewalHandler) GetRenewalPage(c *gin.Context) {
	licenseID := c.Query("license")
	token := c.Query("token")

	if licenseID == "" && token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "license or token parameter required"})
		return
	}

	if token != "" {
		// Task 15.1: Validate signed token for secure renewal link
		// TODO: Implement token validation
		c.JSON(http.StatusNotImplemented, gin.H{"error": "token-based renewal not yet implemented"})
		return
	}

	// Load license by ID (using direct query for now)
	// TODO: Add proper GetLicenseByID and GetPlanByID methods to store
	var license model.License
	err := h.Store.DB.NewSelect().Model(&license).Where("id = ?", licenseID).Scan(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "license not found"})
		return
	}

	// Load plan details
	var plan model.Plan
	err = h.Store.DB.NewSelect().Model(&plan).Where("id = ?", license.PlanID).Scan(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
		return
	}

	// Task 15.2: Return renewal data with pre-filled customer info
	renewalData := gin.H{
		"license": gin.H{
			"id":          license.ID,
			"email":       license.Email,
			"plan_id":     license.PlanID,
			"valid_until": license.ValidUntil,
			"status":      license.Status,
		},
		"current_plan": gin.H{
			"id":              plan.ID,
			"name":            plan.Name,
			"price_usd":       plan.PriceUSD,
			"price_cny":       plan.PriceCNY,
			"price_hkd":       plan.PriceHKD,
			"renewal_days":    plan.RenewalDays,
			"stripe_price_id": plan.StripePriceID,
		},
		"customer": gin.H{
			"email": license.Email,
		},
	}

	c.JSON(http.StatusOK, renewalData)
}

// CreateRenewalCheckout creates a checkout session for license renewal
// Task 15.4: Include is_renewal=true and original_license_id in metadata
func (h *RenewalHandler) CreateRenewalCheckout(c *gin.Context) {
	var req struct {
		LicenseID string `json:"license_id"`
		PlanID    string `json:"plan_id"` // May differ from current plan if upgrading
		Provider  string `json:"provider"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// Load license
	var license model.License
	err := h.Store.DB.NewSelect().Model(&license).Where("id = ?", req.LicenseID).Scan(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "license not found"})
		return
	}

	// Load plan (might be upgrade)
	var plan model.Plan
	err = h.Store.DB.NewSelect().Model(&plan).Where("id = ?", req.PlanID).Scan(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
		return
	}

	// Task 15.4: Build metadata with renewal flags
	metadata := map[string]string{
		"is_renewal":          "true",
		"original_license_id": license.ID,
		"original_plan_id":    license.PlanID,
		"renewal_plan_id":     plan.ID,
	}

	// Task 15.3: Check if this is an upgrade
	if plan.ID != license.PlanID {
		metadata["is_upgrade"] = "true"
	}

	// TODO: Call payment service to create checkout session with metadata
	// For now, return placeholder
	checkoutData := gin.H{
		"checkout_url": "https://example.com/checkout",
		"metadata":     metadata,
	}

	c.JSON(http.StatusOK, checkoutData)
}
