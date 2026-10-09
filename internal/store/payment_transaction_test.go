package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/tabloy/keygate/internal/model"
)

func TestPaymentTransaction_CreateAndLookup(t *testing.T) {
	s := setupTestDB(t)
	ctx := context.Background()
	lic := createTestLicense(t, s, ctx)

	txn := &model.PaymentTransaction{
		ProviderName:  "alipay",
		ProviderTxID:  "tx-" + lic.ID,
		LicenseID:     lic.ID,
		Amount:        9900,
		Currency:      "CNY",
		Status:        "completed",
		PaymentMethod: "alipay",
		CustomerEmail: lic.Email,
		Metadata:      "{}",
		ProcessedAt:   time.Now(),
	}
	if err := s.CreatePaymentTransaction(ctx, txn); err != nil {
		t.Fatalf("create: %v", err)
	}
	if txn.ID == 0 {
		t.Fatal("expected ID to be populated via RETURNING")
	}

	exists, err := s.PaymentTransactionExists(ctx, "alipay", txn.ProviderTxID)
	if err != nil || !exists {
		t.Fatalf("exists = %v, err = %v", exists, err)
	}

	got, err := s.GetPaymentTransactionByProviderID(ctx, "alipay", txn.ProviderTxID)
	if err != nil || got == nil {
		t.Fatalf("get by provider: %v, %v", got, err)
	}
	if got.LicenseID != lic.ID || got.Amount != 9900 {
		t.Fatalf("unexpected row: %+v", got)
	}

	byLicense, err := s.GetPaymentTransactionsByLicense(ctx, lic.ID)
	if err != nil || len(byLicense) != 1 {
		t.Fatalf("by license: %d rows, err = %v", len(byLicense), err)
	}

	got.Status = "refunded"
	if err := s.UpdatePaymentTransaction(ctx, got); err != nil {
		t.Fatalf("update: %v", err)
	}

	missing, err := s.GetPaymentTransactionByProviderID(ctx, "alipay", "does-not-exist")
	if err != nil || missing != nil {
		t.Fatalf("missing lookup: %v, %v", missing, err)
	}
}

func TestExtendLicense_AddsDays(t *testing.T) {
	s := setupTestDB(t)
	ctx := context.Background()
	lic := createTestLicense(t, s, ctx)

	start := time.Now().Add(10 * 24 * time.Hour).UTC().Truncate(time.Second)
	lic.ValidUntil = &start
	if err := s.UpdateLicense(ctx, lic, "valid_until"); err != nil {
		t.Fatalf("set valid_until: %v", err)
	}

	if err := s.ExtendLicense(ctx, lic.ID, 30); err != nil {
		t.Fatalf("extend: %v", err)
	}

	var got model.License
	if err := s.DB.NewSelect().Model(&got).Where("id = ?", lic.ID).Scan(ctx); err != nil {
		t.Fatalf("reload: %v", err)
	}
	want := start.Add(30 * 24 * time.Hour)
	if got.ValidUntil == nil || !got.ValidUntil.Equal(want) {
		t.Fatalf("valid_until = %v, want %v", got.ValidUntil, want)
	}
}

func TestRenewalReminders_Roundtrip(t *testing.T) {
	s := setupTestDB(t)
	ctx := context.Background()
	lic := createTestLicense(t, s, ctx)

	if _, err := s.FindLicensesExpiringIn(ctx, 7); err != nil {
		t.Fatalf("find expiring: %v", err)
	}

	if err := s.MarkReminderSent(ctx, lic.ID, 7); err != nil {
		t.Fatalf("mark sent: %v", err)
	}
	sent, err := s.WasReminderSent(ctx, lic.ID, 7)
	if err != nil || !sent {
		t.Fatalf("was sent = %v, err = %v", sent, err)
	}
	if err := s.ClearRenewalReminders(ctx, lic.ID); err != nil {
		t.Fatalf("clear: %v", err)
	}
	sent, err = s.WasReminderSent(ctx, lic.ID, 7)
	if err != nil || sent {
		t.Fatalf("after clear: sent = %v, err = %v", sent, err)
	}
}
