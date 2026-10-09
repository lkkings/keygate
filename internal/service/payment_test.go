package service

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/tabloy/keygate/internal/model"
	"github.com/tabloy/keygate/internal/payment"
	"github.com/tabloy/keygate/internal/store"
)

// A renewal webhook extends the license, records the transaction against
// it and clears its reminders; a redelivered webhook changes nothing.
func TestFulfillPayment_RenewalIsIdempotent(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("skipping integration test: TEST_DATABASE_URL not set")
	}
	s, err := store.New(dsn)
	if err != nil {
		t.Skipf("skipping integration test: %v", err)
	}
	defer s.Close()
	if err := s.RunMigrations("../../db/migrations"); err != nil {
		t.Fatalf("migrations: %v", err)
	}
	ctx := context.Background()
	suffix := time.Now().Format("150405.000000")

	prod := &model.Product{Name: "Renewal", Slug: "renewal-" + suffix, Type: "desktop"}
	if err := s.CreateProduct(ctx, prod); err != nil {
		t.Fatal(err)
	}
	plan := &model.Plan{ProductID: prod.ID, Name: "Yearly", Slug: "renewal-" + suffix, LicenseType: "subscription", LicenseModel: "standard"}
	if err := s.CreatePlan(ctx, plan); err != nil {
		t.Fatal(err)
	}
	expiry := time.Now().Add(7 * 24 * time.Hour).UTC().Truncate(time.Second)
	lic := &model.License{
		ProductID: prod.ID, PlanID: plan.ID, Email: "renew-" + suffix + "@example.com",
		LicenseKey: "RENEW-" + suffix, Status: model.StatusActive, ValidUntil: &expiry,
	}
	if err := s.CreateLicense(ctx, lic); err != nil {
		t.Fatal(err)
	}
	if err := s.MarkReminderSent(ctx, lic.ID, 7); err != nil {
		t.Fatal(err)
	}

	svc := NewPaymentService(s, nil, nil, nil)
	event := payment.WebhookEvent{
		Provider:            "alipay",
		TransactionID:       "renew-tx-" + suffix,
		RenewalForLicenseID: lic.ID,
		ExtensionDays:       365,
		Amount:              9900,
		Currency:            "CNY",
		CustomerEmail:       lic.Email,
	}

	for i := 0; i < 2; i++ {
		if err := svc.FulfillPayment(ctx, "alipay", event); err != nil {
			t.Fatalf("fulfill #%d: %v", i+1, err)
		}
	}

	var got model.License
	if err := s.DB.NewSelect().Model(&got).Where("id = ?", lic.ID).Scan(ctx); err != nil {
		t.Fatal(err)
	}
	want := expiry.Add(365 * 24 * time.Hour)
	if got.ValidUntil == nil || !got.ValidUntil.Equal(want) {
		t.Fatalf("valid_until = %v, want %v (extended exactly once)", got.ValidUntil, want)
	}

	txns, err := s.GetPaymentTransactionsByLicense(ctx, lic.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(txns) != 1 || txns[0].Status != "completed" {
		t.Fatalf("transactions = %+v, want one completed", txns)
	}

	sent, err := s.WasReminderSent(ctx, lic.ID, 7)
	if err != nil {
		t.Fatal(err)
	}
	if sent {
		t.Fatal("renewal should clear reminder records")
	}
}
