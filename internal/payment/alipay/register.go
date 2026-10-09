package alipay

import (
	"github.com/tabloy/keygate/internal/payment"
)

func init() {
	// Registration happens during app initialization
	// when config is loaded, not in init()
}

// Register registers the Alipay provider with the payment registry
// This should be called during app initialization with the configured provider
func Register(provider *Provider) error {
	return payment.RegisterProvider(provider)
}
