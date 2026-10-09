package payment

import "fmt"

// ErrProviderNotFound is returned when a requested payment provider is not registered
type ErrProviderNotFound struct {
	Provider string
}

func (e *ErrProviderNotFound) Error() string {
	return fmt.Sprintf("payment provider not found: %s", e.Provider)
}

// ErrProviderDisabled is returned when a payment provider is disabled in settings
type ErrProviderDisabled struct {
	Provider string
}

func (e *ErrProviderDisabled) Error() string {
	return fmt.Sprintf("payment provider is disabled: %s", e.Provider)
}

// ErrInvalidSignature is returned when webhook signature verification fails
type ErrInvalidSignature struct {
	Provider string
	Reason   string
}

func (e *ErrInvalidSignature) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("invalid signature for %s: %s", e.Provider, e.Reason)
	}
	return fmt.Sprintf("invalid signature for %s", e.Provider)
}

// ErrPaymentNotFound is returned when a payment transaction cannot be found
type ErrPaymentNotFound struct {
	TransactionID string
	Provider      string
}

func (e *ErrPaymentNotFound) Error() string {
	return fmt.Sprintf("payment not found: %s (provider: %s)", e.TransactionID, e.Provider)
}

// ErrInvalidAmount is returned when payment amount is invalid
type ErrInvalidAmount struct {
	Amount   int64
	Currency string
	Reason   string
}

func (e *ErrInvalidAmount) Error() string {
	return fmt.Sprintf("invalid amount %d %s: %s", e.Amount, e.Currency, e.Reason)
}

// ErrRefundFailed is returned when a refund operation fails
type ErrRefundFailed struct {
	TransactionID string
	Provider      string
	Reason        string
}

func (e *ErrRefundFailed) Error() string {
	return fmt.Sprintf("refund failed for %s (%s): %s", e.TransactionID, e.Provider, e.Reason)
}

// ErrCheckoutFailed is returned when checkout creation fails
type ErrCheckoutFailed struct {
	Provider string
	Reason   string
}

func (e *ErrCheckoutFailed) Error() string {
	return fmt.Sprintf("checkout creation failed for %s: %s", e.Provider, e.Reason)
}

// ErrNotImplemented is returned for features not yet implemented
type ErrNotImplemented struct {
	Provider string
	Feature  string
}

func (e *ErrNotImplemented) Error() string {
	return fmt.Sprintf("%s: %s is not implemented yet", e.Provider, e.Feature)
}
