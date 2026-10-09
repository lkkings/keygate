package store

import (
	"context"
	"fmt"
	"strings"
)

// PaymentProviderConfig holds configuration for a payment provider
type PaymentProviderConfig struct {
	Enabled    bool
	Settings   map[string]string
}

// GetPaymentProviderConfig retrieves all settings for a specific provider
func (s *Store) GetPaymentProviderConfig(ctx context.Context, provider string) (*PaymentProviderConfig, error) {
	allSettings, err := s.GetSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}

	config := &PaymentProviderConfig{
		Settings: make(map[string]string),
	}

	// Determine provider prefix
	prefix := provider + "_"

	// Extract provider-specific settings
	for key, value := range allSettings {
		if strings.HasPrefix(key, prefix) {
			// Store the full key
			config.Settings[key] = value

			// Check enabled flag
			if key == prefix+"enabled" {
				config.Enabled = (value == "true" || value == "1")
			}
		}
	}

	return config, nil
}

// SetPaymentProviderConfig saves all settings for a specific provider
func (s *Store) SetPaymentProviderConfig(ctx context.Context, provider string, settings map[string]string) error {
	// Validate that all keys have the correct prefix
	prefix := provider + "_"
	for key := range settings {
		if !strings.HasPrefix(key, prefix) {
			return fmt.Errorf("invalid setting key %s: must start with %s", key, prefix)
		}
	}

	return s.SetSettings(ctx, settings)
}

// GetPaymentProviderSetting retrieves a single setting for a provider
func (s *Store) GetPaymentProviderSetting(ctx context.Context, provider, key string) (string, error) {
	fullKey := provider + "_" + key
	return s.GetSetting(ctx, fullKey)
}

// SetPaymentProviderSetting saves a single setting for a provider
func (s *Store) SetPaymentProviderSetting(ctx context.Context, provider, key, value string) error {
	fullKey := provider + "_" + key
	return s.SetSetting(ctx, fullKey, value)
}

// IsPaymentProviderEnabled checks if a payment provider is enabled
func (s *Store) IsPaymentProviderEnabled(ctx context.Context, provider string) (bool, error) {
	value, err := s.GetPaymentProviderSetting(ctx, provider, "enabled")
	if err != nil {
		return false, err
	}
	return value == "true" || value == "1", nil
}
