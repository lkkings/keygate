package stripe

import (
	"github.com/tabloy/keygate/internal/payment"
)

// Register registers the Stripe provider with the payment registry
// This should be called during app initialization with the configured provider
func Register(provider *Provider) error {
	return payment.RegisterProvider(provider)
}
