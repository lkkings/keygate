package payment

import (
	"context"
	"errors"
	"sort"
	"testing"
)

type fakeProvider struct{ name string }

func (f *fakeProvider) Name() string { return f.name }
func (f *fakeProvider) CreateCheckout(context.Context, CheckoutParams) (CheckoutResult, error) {
	return CheckoutResult{}, nil
}
func (f *fakeProvider) VerifyWebhook(context.Context, []byte, string) (WebhookEvent, error) {
	return WebhookEvent{}, nil
}
func (f *fakeProvider) RefundPayment(context.Context, string, int64, string) (RefundResult, error) {
	return RefundResult{}, nil
}
func (f *fakeProvider) GetPaymentStatus(context.Context, string) (PaymentStatus, error) {
	return PaymentStatus{}, nil
}

func TestProviderRegistry(t *testing.T) {
	ClearRegistry()
	t.Cleanup(ClearRegistry)

	if err := RegisterProvider(&fakeProvider{name: "stripe"}); err != nil {
		t.Fatalf("register stripe: %v", err)
	}
	if err := RegisterProvider(&fakeProvider{name: "alipay"}); err != nil {
		t.Fatalf("register alipay: %v", err)
	}

	tests := []struct {
		name    string
		run     func() error
		wantErr bool
	}{
		{"nil provider rejected", func() error { return RegisterProvider(nil) }, true},
		{"empty name rejected", func() error { return RegisterProvider(&fakeProvider{}) }, true},
		{"duplicate rejected", func() error { return RegisterProvider(&fakeProvider{name: "stripe"}) }, true},
		{"get registered", func() error { _, err := GetProvider("stripe"); return err }, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	t.Run("get unknown returns ErrProviderNotFound", func(t *testing.T) {
		_, err := GetProvider("paypal")
		var notFound *ErrProviderNotFound
		if !errors.As(err, &notFound) {
			t.Fatalf("err = %v, want ErrProviderNotFound", err)
		}
	})

	t.Run("list and filter enabled", func(t *testing.T) {
		all := ListProviders()
		sort.Strings(all)
		if len(all) != 2 || all[0] != "alipay" || all[1] != "stripe" {
			t.Fatalf("ListProviders = %v", all)
		}
		enabled := ListEnabledProviders(map[string]bool{"stripe": true, "alipay": false, "wechat": true})
		if len(enabled) != 1 || enabled[0] != "stripe" {
			t.Fatalf("ListEnabledProviders = %v", enabled)
		}
	})

	t.Run("unregister", func(t *testing.T) {
		UnregisterProvider("alipay")
		if _, err := GetProvider("alipay"); err == nil {
			t.Fatal("expected alipay to be gone")
		}
	})
}
