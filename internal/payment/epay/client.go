package epay

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// Client handles ePay API interactions
// NOTE: This is a stub implementation. ePay API documentation is not available.
// The actual implementation will need to be completed once official API specs are obtained.
type Client struct {
	MerchantID string
	APIKey     string
	Gateway    string
	HTTPClient *http.Client
}

// Config holds ePay client configuration
type Config struct {
	MerchantID string
	APIKey     string
	Gateway    string // Custom gateway URL if provided
	IsSandbox  bool
}

// NewClient creates a new ePay client
func NewClient(config Config) (*Client, error) {
	gateway := config.Gateway
	if gateway == "" {
		// TODO: Replace with actual ePay production gateway once known
		gateway = "https://api.epay.example.com"
		if config.IsSandbox {
			gateway = "https://sandbox.epay.example.com"
		}
	}

	return &Client{
		MerchantID: config.MerchantID,
		APIKey:     config.APIKey,
		Gateway:    gateway,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// CreateOrder creates a new payment order
// TODO: Implement once ePay API specification is available
func (c *Client) CreateOrder(ctx context.Context, params *CreateOrderRequest) (*CreateOrderResponse, error) {
	return nil, fmt.Errorf("ePay CreateOrder not implemented: awaiting official API documentation")
}

// VerifyNotification verifies a payment notification signature
// TODO: Implement once ePay signature algorithm is confirmed
func (c *Client) VerifyNotification(params map[string]string) error {
	return fmt.Errorf("ePay VerifyNotification not implemented: awaiting signature algorithm specification")
}

// QueryOrder queries the status of an order
// TODO: Implement once ePay query API is documented
func (c *Client) QueryOrder(ctx context.Context, orderID string) (*QueryOrderResponse, error) {
	return nil, fmt.Errorf("ePay QueryOrder not implemented: awaiting official API documentation")
}

// RefundOrder initiates a refund
// TODO: Implement once ePay refund API is documented
func (c *Client) RefundOrder(ctx context.Context, params *RefundRequest) (*RefundResponse, error) {
	return nil, fmt.Errorf("ePay RefundOrder not implemented: awaiting official API documentation")
}

// Request and Response types
// NOTE: These are placeholder structures and will need to be updated
// based on actual ePay API documentation

// CreateOrderRequest represents an order creation request
type CreateOrderRequest struct {
	OrderID     string
	Amount      int64
	Currency    string
	Description string
	NotifyURL   string
	ReturnURL   string
	// TODO: Add actual ePay required fields
}

// CreateOrderResponse represents an order creation response
type CreateOrderResponse struct {
	OrderID     string
	PaymentURL  string
	QRCodeURL   string
	Status      string
	// TODO: Add actual ePay response fields
}

// QueryOrderRequest represents an order query request
type QueryOrderRequest struct {
	OrderID string
	// TODO: Add actual ePay required fields
}

// QueryOrderResponse represents an order query response
type QueryOrderResponse struct {
	OrderID       string
	Status        string
	Amount        int64
	Currency      string
	TransactionID string
	PaidAt        string
	// TODO: Add actual ePay response fields
}

// RefundRequest represents a refund request
type RefundRequest struct {
	OrderID     string
	RefundID    string
	Amount      int64
	Reason      string
	// TODO: Add actual ePay required fields
}

// RefundResponse represents a refund response
type RefundResponse struct {
	RefundID      string
	Status        string
	Amount        int64
	TransactionID string
	// TODO: Add actual ePay response fields
}
