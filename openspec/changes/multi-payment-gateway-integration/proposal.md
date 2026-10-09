# Multi-Payment Gateway Integration

## Why

Keygate currently only supports Stripe as a payment provider, limiting its usability in markets where Alipay, WeChat Pay, and other regional payment methods dominate. To expand market reach, particularly in China and Asia-Pacific regions, we need to integrate multiple payment gateways with a unified abstraction layer that allows customers to choose their preferred payment method during checkout.

## What Changes

- Add payment provider abstraction layer with `PaymentProvider` interface
- Integrate **Alipay** (支付宝) for web and QR code payments
- Integrate **WeChat Pay** (微信支付) for JSAPI, Native (QR code), and App payments
- Integrate **ePay** (易派支付) for additional payment options
- Extend `plans` table to support multi-currency pricing (USD, CNY, HKD, etc.)
- Add payment configuration UI in admin settings with per-provider credentials and webhook URLs
- Create `payment_transactions` table to track all payment events across providers
- Extend `licenses` table with provider-agnostic payment reference fields
- Implement subscription renewal reminder system (since Alipay/WeChat don't support native subscriptions)
- Add provider-specific webhook endpoints: `/api/v1/payment/:provider/webhook`
- Maintain backward compatibility with existing Stripe integration

## Capabilities

### New Capabilities

- `payment-providers`: Payment gateway abstraction layer and provider registry
- `alipay-integration`: Alipay payment provider implementation with signature verification
- `wechat-pay-integration`: WeChat Pay provider with JSAPI, Native, and App payment modes
- `epay-integration`: ePay payment provider integration
- `multi-currency-pricing`: Multi-currency price configuration in plans (price_cny, price_hkd, etc.)
- `subscription-renewal-reminders`: Automated renewal reminders for non-subscription payment methods
- `payment-settings-ui`: Admin UI for configuring payment provider credentials and testing connections

### Modified Capabilities

<!-- No existing capabilities require spec-level changes. The Stripe integration remains unchanged;
     new providers are additive. -->

## Impact

### Backend (Go)

**New Files:**
- `internal/payment/provider.go` - PaymentProvider interface
- `internal/payment/provider_registry.go` - Provider registration and routing
- `internal/payment/alipay/` - Alipay provider implementation
- `internal/payment/wechat/` - WeChat Pay provider implementation
- `internal/payment/epay/` - ePay provider implementation
- `internal/handler/payment.go` - Generic payment route handlers

**Modified Files:**
- `internal/model/model.go` - Extend Plan with multi-currency prices
- `internal/store/settings.go` - Add payment provider settings accessors
- `cmd/server/main.go` - Register new payment routes

**Database:**
- New table: `payment_transactions` (provider-agnostic transaction log)
- Extend `licenses`: add `payment_reference`, `payment_metadata` columns
- Extend `plans`: add `price_cny`, `price_hkd`, `stripe_price_id_cny`, etc.
- Extend `settings`: add keys for Alipay/WeChat/ePay credentials

### Frontend (React/TypeScript)

**New Files:**
- `web/src/pages/admin/payment-settings.tsx` - Payment provider configuration
- `web/src/components/payment-method-selector.tsx` - Customer-facing payment method picker
- `web/src/lib/payment-api.ts` - Payment API client

**Modified Files:**
- `web/src/pages/admin/settings.tsx` - Add Payment tab
- `web/src/pages/admin/plans.tsx` - Add multi-currency price fields

### APIs

**New Endpoints:**
- `POST /api/v1/payment/:provider/checkout` - Create payment session
- `POST /api/v1/payment/:provider/webhook` - Handle provider webhooks
- `GET /api/v1/payment/:provider/verify` - Verify payment status
- `POST /api/v1/payment/:provider/refund` - Process refunds

**Dependencies:**
- Alipay SDK (if official Go SDK exists, or implement signature logic)
- WeChat Pay SDK or direct API integration
- ePay SDK/API integration

### Migration Path

1. Deploy backend with new payment providers disabled by default
2. Configure payment providers through admin UI
3. Test each provider in sandbox/test mode
4. Enable providers in production settings
5. Existing Stripe integrations continue working unchanged
