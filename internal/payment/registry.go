package payment

import (
	"fmt"
	"sync"
)

var (
	// Global registry of payment providers
	providerRegistry = &Registry{
		providers: make(map[string]PaymentProvider),
	}
)

// Registry manages registered payment providers
type Registry struct {
	mu        sync.RWMutex
	providers map[string]PaymentProvider
}

// RegisterProvider registers a payment provider
func RegisterProvider(provider PaymentProvider) error {
	if provider == nil {
		return fmt.Errorf("provider cannot be nil")
	}

	name := provider.Name()
	if name == "" {
		return fmt.Errorf("provider name cannot be empty")
	}

	providerRegistry.mu.Lock()
	defer providerRegistry.mu.Unlock()

	if _, exists := providerRegistry.providers[name]; exists {
		return fmt.Errorf("provider %s is already registered", name)
	}

	providerRegistry.providers[name] = provider
	return nil
}

// GetProvider retrieves a payment provider by name
func GetProvider(name string) (PaymentProvider, error) {
	providerRegistry.mu.RLock()
	defer providerRegistry.mu.RUnlock()

	provider, exists := providerRegistry.providers[name]
	if !exists {
		return nil, &ErrProviderNotFound{Provider: name}
	}

	return provider, nil
}

// ListProviders returns all registered provider names
func ListProviders() []string {
	providerRegistry.mu.RLock()
	defer providerRegistry.mu.RUnlock()

	names := make([]string, 0, len(providerRegistry.providers))
	for name := range providerRegistry.providers {
		names = append(names, name)
	}
	return names
}

// ListEnabledProviders returns names of enabled providers based on settings
// This function should be called with the current settings to filter enabled providers
func ListEnabledProviders(enabledMap map[string]bool) []string {
	providerRegistry.mu.RLock()
	defer providerRegistry.mu.RUnlock()

	enabled := make([]string, 0)
	for name := range providerRegistry.providers {
		if isEnabled, exists := enabledMap[name]; exists && isEnabled {
			enabled = append(enabled, name)
		}
	}
	return enabled
}

// UnregisterProvider removes a provider from the registry (useful for testing)
func UnregisterProvider(name string) {
	providerRegistry.mu.Lock()
	defer providerRegistry.mu.Unlock()
	delete(providerRegistry.providers, name)
}

// ClearRegistry clears all registered providers (useful for testing)
func ClearRegistry() {
	providerRegistry.mu.Lock()
	defer providerRegistry.mu.Unlock()
	providerRegistry.providers = make(map[string]PaymentProvider)
}
