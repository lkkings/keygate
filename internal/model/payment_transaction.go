package model

import (
	"time"

	"github.com/uptrace/bun"
)

// PaymentTransaction represents a payment transaction record
type PaymentTransaction struct {
	bun.BaseModel `bun:"table:payment_transactions"`

	ID            int64  `bun:",pk,autoincrement" json:"id"`
	ProviderName  string `bun:",notnull" json:"provider_name"`
	ProviderTxID  string `bun:",notnull" json:"provider_tx_id"`
	SessionID     string `bun:",notnull" json:"session_id"`
	LicenseID     string `bun:",nullzero" json:"license_id,omitempty"`
	Amount        int64  `bun:",notnull" json:"amount"` // Amount in cents
	Currency      string `bun:",notnull" json:"currency"`
	Status        string `bun:",notnull" json:"status"`         // pending, completed, failed, cancelled, refunded
	PaymentMethod string `bun:",notnull" json:"payment_method"` // stripe, alipay, wechat, etc.
	CustomerEmail string `bun:",notnull" json:"customer_email"`
	// Task 16.4: Refund fields
	RefundAmount int64      `bun:",notnull" json:"refund_amount,omitempty"`
	RefundedAt   *time.Time `json:"refunded_at,omitempty"`
	Metadata     string     `bun:",notnull" json:"metadata"` // JSON string
	ProcessedAt  time.Time  `bun:",notnull,default:current_timestamp" json:"processed_at"`
	CreatedAt    time.Time  `bun:",notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt    time.Time  `bun:",notnull,default:current_timestamp" json:"updated_at"`
}
