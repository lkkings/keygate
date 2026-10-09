package handler

import "testing"

func TestPaymentWebhookBaseURL(t *testing.T) {
	tests := []struct {
		name       string
		configured string
		inferred   string
		want       string
	}{
		{"configured https kept", "https://pay.example.com", "http://ignored", "https://pay.example.com"},
		{"configured http upgraded", "http://pay.example.com/", "", "https://pay.example.com"},
		{"falls back to inferred", "", "http://10.0.0.5:9000", "https://10.0.0.5:9000"},
		{"inferred https kept", "  ", "https://keygate.example.com", "https://keygate.example.com"},
		{"bare host gets scheme", "keygate.example.com", "", "https://keygate.example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := paymentWebhookBaseURL(tt.configured, tt.inferred); got != tt.want {
				t.Errorf("paymentWebhookBaseURL(%q, %q) = %q, want %q", tt.configured, tt.inferred, got, tt.want)
			}
		})
	}
}
