package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// GetPaymentSettings returns all payment provider settings with secrets masked
// GET /api/v1/admin/settings/payment
func (h *AdminHandler) GetPaymentSettings(c *gin.Context) {
	ctx := c.Request.Context()

	// Get all settings
	allSettings, err := h.Store.GetSettings(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get settings"})
		return
	}

	// Group by provider and mask secrets
	providers := make(map[string]map[string]interface{})
	providerNames := []string{"stripe", "alipay", "wechat", "epay"}

	for _, provider := range providerNames {
		providerSettings := make(map[string]interface{})
		prefix := provider + "_"

		for key, value := range allSettings {
			if strings.HasPrefix(key, prefix) {
				// Remove provider prefix for cleaner response
				shortKey := strings.TrimPrefix(key, prefix)

				// Mask secrets
				if isSecretField(shortKey) && value != "" {
					providerSettings[shortKey] = maskSecret(value)
				} else {
					providerSettings[shortKey] = value
				}
			}
		}

		// Only include if provider has settings
		if len(providerSettings) > 0 {
			providers[provider] = providerSettings
		}
	}

	baseURL := paymentWebhookBaseURL(h.BaseURL, adminBaseURL(c))
	webhookURLs := make(map[string]string)
	for _, provider := range providerNames {
		webhookURLs[provider] = baseURL + "/api/v1/payment/" + provider + "/webhook"
	}

	c.JSON(http.StatusOK, gin.H{
		"providers":    providers,
		"webhook_urls": webhookURLs,
	})
}

// UpdatePaymentProviderSettings updates settings for a specific provider
// PUT /api/v1/admin/settings/payment/:provider
func (h *AdminHandler) UpdatePaymentProviderSettings(c *gin.Context) {
	ctx := c.Request.Context()
	provider := c.Param("provider")

	// Validate provider name
	validProviders := map[string]bool{
		"stripe": true,
		"alipay": true,
		"wechat": true,
		"epay":   true,
	}
	if !validProviders[provider] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid provider: " + provider})
		return
	}

	// Parse request body
	var req map[string]string
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Get existing settings to preserve secrets if not provided
	existingConfig, err := h.Store.GetPaymentProviderConfig(ctx, provider)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get existing settings"})
		return
	}

	// Build settings map with provider prefix
	settings := make(map[string]string)
	for key, value := range req {
		fullKey := provider + "_" + key

		// If value is masked placeholder and we have existing value, preserve it
		if isMaskedValue(value) && existingConfig.Settings[fullKey] != "" {
			settings[fullKey] = existingConfig.Settings[fullKey]
		} else if value != "" {
			settings[fullKey] = value
		}
	}

	// Save settings
	if err := h.Store.SetPaymentProviderConfig(ctx, provider, settings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save settings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "settings updated"})
}

// TestPaymentProviderConnection tests the connection to a payment provider
// POST /api/v1/admin/settings/payment/:provider/test
func (h *AdminHandler) TestPaymentProviderConnection(c *gin.Context) {
	provider := c.Param("provider")

	// TODO: Implement actual test connection logic
	// This would involve creating a provider instance and making a test API call
	// For now, return not implemented

	c.JSON(http.StatusOK, gin.H{
		"provider": provider,
		"status":   "not_implemented",
		"message":  "Test connection not yet implemented",
	})
}

// Helper functions

// paymentWebhookBaseURL picks the public origin payment providers call
// back: BASE_URL when configured, else the one the admin reached us
// on. Always https — Alipay and WeChat Pay reject plain-HTTP notify
// URLs, and Stripe requires TLS for live-mode endpoints.
func paymentWebhookBaseURL(configured, inferred string) string {
	base := strings.TrimSpace(configured)
	if base == "" {
		base = inferred
	}
	base = strings.TrimRight(base, "/")
	if rest, ok := strings.CutPrefix(base, "http://"); ok {
		return "https://" + rest
	}
	if !strings.HasPrefix(base, "https://") {
		return "https://" + base
	}
	return base
}

// isSecretField checks if a field name indicates secret data
func isSecretField(key string) bool {
	secretKeys := []string{
		"secret_key",
		"private_key",
		"public_key",
		"api_key",
		"api_secret",
		"webhook_secret",
		"cert_file",
		"key_file",
	}

	keyLower := strings.ToLower(key)
	for _, secret := range secretKeys {
		if strings.Contains(keyLower, secret) {
			return true
		}
	}
	return false
}

// maskSecret masks a secret value for display
func maskSecret(value string) string {
	if len(value) <= 8 {
		return "********"
	}
	// Show first 4 and last 4 characters
	return value[:4] + "..." + value[len(value)-4:]
}

// isMaskedValue checks if a value is a masked placeholder
func isMaskedValue(value string) bool {
	return strings.Contains(value, "***") || strings.Contains(value, "...")
}
