package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/tabloy/keygate/internal/model"
)

// CreatePaymentTransaction creates a new payment transaction record.
// tx.ID is populated from the database via RETURNING.
func (s *Store) CreatePaymentTransaction(ctx context.Context, tx *model.PaymentTransaction) error {
	if _, err := s.DB.NewInsert().Model(tx).
		ExcludeColumn("id", "created_at", "updated_at").
		Returning("id, created_at, updated_at").
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to create payment transaction: %w", err)
	}
	return nil
}

// GetPaymentTransaction retrieves a payment transaction by its internal ID
func (s *Store) GetPaymentTransaction(ctx context.Context, id string) (*model.PaymentTransaction, error) {
	tx := new(model.PaymentTransaction)
	err := s.DB.NewSelect().Model(tx).Where("id = ?", id).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get payment transaction: %w", err)
	}
	return tx, nil
}

// GetPaymentTransactionByProviderID retrieves a payment transaction by provider and provider transaction ID
func (s *Store) GetPaymentTransactionByProviderID(ctx context.Context, provider, providerTxID string) (*model.PaymentTransaction, error) {
	tx := new(model.PaymentTransaction)
	err := s.DB.NewSelect().Model(tx).
		Where("provider_name = ? AND provider_tx_id = ?", provider, providerTxID).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get payment transaction: %w", err)
	}
	return tx, nil
}

// UpdatePaymentTransaction updates a payment transaction
func (s *Store) UpdatePaymentTransaction(ctx context.Context, tx *model.PaymentTransaction) error {
	if _, err := s.DB.NewUpdate().Model(tx).
		Column("status", "payment_method", "metadata").
		Set("updated_at = now()").
		WherePK().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to update payment transaction: %w", err)
	}
	return nil
}

// GetPaymentTransactionsByEmail retrieves all payment transactions for a customer email
func (s *Store) GetPaymentTransactionsByEmail(ctx context.Context, email string) ([]*model.PaymentTransaction, error) {
	var transactions []*model.PaymentTransaction
	if err := s.DB.NewSelect().Model(&transactions).
		Where("customer_email = ?", email).
		Order("created_at DESC").
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("failed to query payment transactions: %w", err)
	}
	return transactions, nil
}

// GetPaymentTransactionsByLicense retrieves all payment transactions for a license
func (s *Store) GetPaymentTransactionsByLicense(ctx context.Context, licenseID string) ([]*model.PaymentTransaction, error) {
	var transactions []*model.PaymentTransaction
	if err := s.DB.NewSelect().Model(&transactions).
		Where("license_id = ?", licenseID).
		Order("created_at DESC").
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("failed to query payment transactions: %w", err)
	}
	return transactions, nil
}

// PaymentTransactionExists checks if a payment transaction already exists (idempotency check)
// Task 13.3: Idempotency based on (provider, provider_transaction_id) uniqueness
func (s *Store) PaymentTransactionExists(ctx context.Context, provider, providerTxID string) (bool, error) {
	exists, err := s.DB.NewSelect().Model((*model.PaymentTransaction)(nil)).
		Where("provider_name = ? AND provider_tx_id = ?", provider, providerTxID).
		Exists(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to check payment transaction existence: %w", err)
	}
	return exists, nil
}
