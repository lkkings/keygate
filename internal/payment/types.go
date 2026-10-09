package payment

import "time"

// CheckoutSession represents a payment checkout session
type CheckoutSession struct {
	ID              string
	Provider        string
	Status          string // "open", "complete", "expired"
	Mode            string // "payment" or "subscription"
	Amount          int64
	Currency        string
	CustomerEmail   string
	SuccessURL      string
	CancelURL       string
	CheckoutURL     string
	QRCodeURL       string // For QR-based payments
	ExpiresAt       time.Time
	Metadata        map[string]string
	CreatedAt       time.Time
}

// CheckoutCustomer represents customer information for checkout
type CheckoutCustomer struct {
	Email       string
	Name        string
	Phone       string
	Address     *CustomerAddress
}

// CustomerAddress represents a customer's address
type CustomerAddress struct {
	Line1      string
	Line2      string
	City       string
	State      string
	PostalCode string
	Country    string
}

// TransactionStatus represents the status of a payment transaction
type TransactionStatus string

const (
	TransactionStatusPending   TransactionStatus = "pending"
	TransactionStatusCompleted TransactionStatus = "completed"
	TransactionStatusFailed    TransactionStatus = "failed"
	TransactionStatusRefunded  TransactionStatus = "refunded"
	TransactionStatusCancelled TransactionStatus = "cancelled"
)

// PaymentMode represents the mode of a payment
type PaymentMode string

const (
	PaymentModePayment      PaymentMode = "payment"
	PaymentModeSubscription PaymentMode = "subscription"
)

// Currency represents supported currencies
type Currency string

const (
	CurrencyUSD Currency = "USD"
	CurrencyCNY Currency = "CNY"
	CurrencyHKD Currency = "HKD"
)

// ProviderName represents supported payment providers
type ProviderName string

const (
	ProviderStripe  ProviderName = "stripe"
	ProviderAlipay  ProviderName = "alipay"
	ProviderWechat  ProviderName = "wechat"
	ProviderEpay    ProviderName = "epay"
)
