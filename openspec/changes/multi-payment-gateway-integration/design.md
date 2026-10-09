# Multi-Payment Gateway Integration - Design

## Context

Keygate currently has a tightly-coupled Stripe integration (`internal/payment/stripe.go`, ~3000 lines) that handles checkout sessions, subscriptions, webhooks, and fulfillment. The existing architecture directly references Stripe SDK types throughout the license fulfillment flow.

**Current Architecture:**
- Single `StripeHandler` struct with methods for each webhook event type
- License fulfillment logic embedded in Stripe-specific handlers
- Settings stored as flat key-value pairs in `settings` table
- Frontend hardcodes Stripe as the only payment option

**Constraints:**
- Must maintain 100% backward compatibility with existing Stripe integration
- Cannot break existing licenses or active subscriptions
- Database schema changes must be backward-compatible (nullable columns)
- Zero downtime deployment requirement

See proposal.md for business motivation.

## Goals / Non-Goals

**Goals:**
- Create payment provider abstraction that isolates gateway-specific logic
- Support Alipay, WeChat Pay, and ePay alongside Stripe
- Enable multi-currency pricing (USD, CNY, HKD) at plan level
- Provide unified admin UI for configuring all payment providers
- Implement renewal reminders for non-subscription payment methods

**Non-Goals:**
- Currency conversion or exchange rate calculations (operators use external tools)
- Payment provider routing/optimization (customer explicitly chooses provider)
- Webhook retry logic beyond what each provider SDK offers
- Subscription management for Alipay/WeChat (manual renewal only)
- Real-time currency switching in checkout UI (currency determined by provider choice)

## Decisions

### Decision 1: Interface-Based Provider Abstraction

**Choice:** Define a `PaymentProvider` interface with methods: `CreateCheckout`, `VerifyWebhook`, `FulfillPayment`, `RefundPayment`, `GetPaymentStatus`.

**Rationale:**
- Decouples payment logic from specific gateway implementations
- Allows testing providers independently via mocks
- Enables adding new providers without modifying core fulfillment logic

**Alternatives Considered:**
- **Strategy pattern with concrete base class:** Go doesn't have inheritance; interfaces are idiomatic
- **Event-driven with message queue:** Over-engineered for 4 providers; adds operational complexity
- **Plugin system with dynamic loading:** Unnecessary complexity; all providers known at compile time

**Implementation:**
```go
type PaymentProvider interface {
    CreateCheckout(ctx context.Context, plan *model.Plan, customer *CheckoutCustomer) (*CheckoutSession, error)
    VerifyWebhook(r *http.Request) (*WebhookEvent, error)
    FulfillPayment(ctx context.Context, event *WebhookEvent) error
    RefundPayment(ctx context.Context, transactionID string, amount int64) error
    GetPaymentStatus(ctx context.Context, transactionID string) (*PaymentStatus, error)
}
```

### Decision 2: Provider Registry Pattern

**Choice:** Implement a global provider registry that maps provider names to implementations.

**Rationale:**
- Enables dynamic routing: `POST /api/v1/payment/:provider/checkout` → registry lookup
- Centralizes provider initialization and configuration loading
- Simplifies testing by allowing mock provider registration

**Implementation:**
```go
// internal/payment/provider_registry.go
var registry = make(map[string]PaymentProvider)

func RegisterProvider(name string, provider PaymentProvider) {
    registry[name] = provider
}

func GetProvider(name string) (PaymentProvider, error) {
    p, ok := registry[name]
    if !ok {
        return nil, fmt.Errorf("unknown provider: %s", name)
    }
    return p, nil
}
```

### Decision 3: Keep Stripe Implementation Unchanged

**Choice:** Wrap existing `StripeHandler` in an adapter that implements `PaymentProvider` interface rather than refactoring it.

**Rationale:**
- Minimizes risk to production Stripe integration (most critical payment flow)
- Avoids regression in battle-tested Stripe webhook handling
- Reduces scope of initial implementation

**Alternatives Considered:**
- **Refactor Stripe into new interface:** High risk, requires extensive testing of existing flows
- **Duplicate Stripe logic:** Code duplication, maintenance burden

**Implementation:**
```go
type StripeProvider struct {
    handler *StripeHandler
}

func (p *StripeProvider) CreateCheckout(ctx context.Context, plan *model.Plan, customer *CheckoutCustomer) (*CheckoutSession, error) {
    // Delegate to existing StripeHandler methods
}
```

### Decision 4: Separate Transaction Log Table

**Choice:** Create new `payment_transactions` table instead of adding provider-specific columns to `licenses`.

**Rationale:**
- Licenses table already has Stripe-specific columns (`stripe_customer_id`, `stripe_subscription_id`)
- Adding columns for each provider would create sparse, fragile schema
- Separate table enables rich transaction history (multiple payments per license)
- Supports future audit requirements and reconciliation

**Schema:**
```sql
CREATE TABLE payment_transactions (
  id TEXT PRIMARY KEY,
  provider TEXT NOT NULL,  -- stripe|alipay|wechat|epay
  provider_transaction_id TEXT NOT NULL,
  license_id TEXT REFERENCES licenses(id),
  plan_id TEXT REFERENCES plans(id),
  amount INTEGER NOT NULL,
  currency TEXT NOT NULL,
  status TEXT NOT NULL,  -- pending|completed|failed|refunded
  payment_method TEXT,
  metadata JSONB,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  UNIQUE(provider, provider_transaction_id)
);
```

**Alternatives Considered:**
- **Expand licenses table:** Would require nullable columns per provider, sparse schema
- **JSONB metadata column in licenses:** Poor queryability, no referential integrity

### Decision 5: Multi-Currency Price Storage

**Choice:** Add dedicated price columns per currency (`price_usd`, `price_cny`, `price_hkd`) to `plans` table.

**Rationale:**
- Simple to query and validate (typed integers)
- Aligns with existing Stripe pattern (separate `stripe_price_id` per currency)
- Avoids JSONB parsing in hot path (checkout creation)

**Alternatives Considered:**
- **JSONB prices column:** Harder to query, no type safety, requires JSON parsing
- **Separate prices table:** Over-normalized for only 3-4 currencies

**Migration:**
```sql
ALTER TABLE plans 
  ADD COLUMN price_cny INTEGER,
  ADD COLUMN price_hkd INTEGER,
  ADD COLUMN stripe_price_id_cny TEXT,
  ADD COLUMN stripe_price_id_hkd TEXT;
```

### Decision 6: Signature Verification Approach

**Choice:** Each provider implements signature verification in its own `VerifyWebhook` method using provider-specific algorithms.

**Rationale:**
- Alipay uses RSA2 (SHA256WithRSA) with public key verification
- WeChat Pay uses MD5 with API key
- Stripe uses HMAC-SHA256 with webhook secret
- No common abstraction possible without loss of security

**Implementation Pattern:**
```go
// internal/payment/alipay/signature.go
func VerifySignature(params map[string]string, publicKey *rsa.PublicKey) error {
    // Sort params, concatenate, verify RSA2 signature
}

// internal/payment/wechat/signature.go
func VerifySignature(params map[string]string, apiKey string) error {
    // Sort params, append key, compute MD5
}
```

### Decision 7: Renewal Reminder as Scheduled Job

**Choice:** Implement renewal reminders as a periodic cron job (daily at 00:00 UTC) that queries expiring licenses.

**Rationale:**
- Simple to implement and monitor
- Acceptable latency (daily check sufficient for 30/14/7/1 day reminders)
- Avoids event-driven complexity

**Alternatives Considered:**
- **Event-driven on license creation:** Requires scheduling system for future events
- **Real-time calculation on email send:** Inefficient, requires scanning all licenses per email

**Implementation:**
```go
func (s *RenewalReminderService) SendDueReminders(ctx context.Context) error {
    for _, days := range []int{30, 14, 7, 1} {
        licenses := s.store.FindLicensesExpiringIn(ctx, days)
        for _, lic := range licenses {
            if !s.wasReminderSent(lic.ID, days) {
                s.sendReminderEmail(lic, days)
                s.markReminderSent(lic.ID, days)
            }
        }
    }
}
```

### Decision 8: Frontend Payment Method Selector

**Choice:** Display payment method options dynamically based on plan's configured currency prices.

**Rationale:**
- Prevents showing Alipay/WeChat if plan has no CNY price
- Provides clear UX (grayed out options with tooltip if unavailable)
- Fetches available providers from backend API (`GET /api/v1/payment/providers?plan_id=xxx`)

**UI Flow:**
1. Customer views plan → Frontend fetches available payment methods for that plan
2. Customer selects payment method (Stripe/Alipay/WeChat/ePay)
3. Frontend calls `POST /api/v1/payment/{provider}/checkout` with plan_id
4. Backend returns redirect URL or QR code data
5. Customer completes payment → Provider webhook → License fulfilled

### Decision 9: Webhook Routing Pattern

**Choice:** Route webhooks via URL path parameter: `/api/v1/payment/:provider/webhook`.

**Rationale:**
- RESTful and predictable (each provider gets its own endpoint)
- Simplifies provider isolation (no parsing webhook body to determine provider)
- Each provider configures its own callback URL in their merchant dashboard

**Implementation:**
```go
func (h *PaymentHandler) HandleWebhook(c *gin.Context) {
    providerName := c.Param("provider")
    provider, err := payment.GetProvider(providerName)
    if err != nil {
        response.NotFound(c, "provider not found")
        return
    }
    event, err := provider.VerifyWebhook(c.Request)
    if err != nil {
        response.Unauthorized(c, "invalid signature")
        return
    }
    if err := provider.FulfillPayment(c.Request.Context(), event); err != nil {
        response.Internal(c, err)
        return
    }
    response.OK(c, gin.H{"received": true})
}
```

### Decision 10: ePay Integration as Placeholder

**Choice:** Implement ePay provider with TODO comments for API-specific details that require official documentation.

**Rationale:**
- ePay API documentation not provided during design phase
- Structure and interface can be defined now based on common payment gateway patterns
- Implementation details filled in once documentation is available

**Documentation Required:**
- ePay API endpoint URLs (production and sandbox)
- Signature algorithm (MD5, HMAC-SHA256, or other)
- Webhook payload format and acknowledgment response
- Supported currencies and payment methods

## Risks / Trade-offs

### Risk 1: Alipay/WeChat No Native Subscription Support
**Risk:** Customers may forget to renew, leading to churn.

**Mitigation:**
- Implement aggressive renewal reminder schedule (30, 14, 7, 1 day before expiration)
- Provide 7-day grace period after expiration
- Include one-click renewal links in all reminder emails
- Monitor renewal conversion rates and adjust reminder frequency

### Risk 2: Signature Verification Bugs
**Risk:** Incorrect signature implementation could accept fraudulent webhooks or reject legitimate ones.

**Mitigation:**
- Test signature verification against official provider test cases
- Log all signature verification failures with full request details (redacting secrets)
- Implement idempotency to prevent duplicate processing if webhook is retried
- Use provider SDKs where available (Stripe) rather than custom implementations

### Risk 3: Currency Price Configuration Errors
**Risk:** Admin may configure inconsistent prices across currencies (e.g., $99 USD but ¥10 CNY).

**Mitigation:**
- Display currency conversion reference in admin UI (informational only, not enforced)
- Validation warning if prices seem inconsistent (>20% deviation from expected exchange rate)
- Allow intentional price localization (emerging markets may warrant different pricing)

### Risk 4: Webhook Replay Attacks
**Risk:** Attacker could replay captured webhook to issue free licenses.

**Mitigation:**
- Verify webhook signature on every request
- Implement idempotency via `processed_events` table (existing Stripe pattern)
- Stripe pattern uses two-row claim system; extend to all providers
- Reject webhooks with timestamps older than 5 minutes (where supported)

### Risk 5: ePay Documentation Incomplete
**Risk:** Cannot fully implement ePay without official API documentation.

**Mitigation:**
- Implement ePay provider structure with interface compatibility
- Mark API-specific sections with TODO comments
- Provide configuration UI skeleton
- Disable ePay in production until implementation is complete and tested

### Risk 6: Database Migration Risk
**Risk:** Adding columns to `plans` and `licenses` tables could lock tables during migration on large datasets.

**Mitigation:**
- All new columns are nullable (no backfill required)
- Migration adds columns without defaults
- Test migration on copy of production database
- Schedule migration during low-traffic window
- Plan rollback: columns can remain unused if rollback needed

### Risk 7: Frontend Payment Method UX Complexity
**Risk:** Showing 4 payment providers could overwhelm customers.

**Mitigation:**
- Display only enabled providers with valid configuration
- Filter by plan's available currencies (hide Alipay if no CNY price)
- Group by region (International: Stripe | China: Alipay, WeChat Pay)
- Implement provider recommendations based on customer's locale

## Migration Plan

### Phase 1: Backend Infrastructure (Week 1-2)
1. Deploy database migrations (add `payment_transactions` table and currency columns)
2. Deploy PaymentProvider interface and registry
3. Deploy Stripe adapter (no behavior change, passes all existing tests)
4. Verify Stripe integration unchanged via existing checkout flow

**Rollback:** Remove provider registry routes, keep Stripe at original URLs.

### Phase 2: Alipay Integration (Week 3-4)
1. Implement Alipay provider with signature verification
2. Add Alipay configuration to settings table
3. Deploy admin UI for Alipay credentials
4. Test in Alipay sandbox environment
5. Enable in production with feature flag

**Rollback:** Disable `alipay_enabled` setting, remove Alipay routes.

### Phase 3: WeChat Pay Integration (Week 5-6)
1. Implement WeChat Pay provider (Native mode first)
2. Add WeChat configuration to settings
3. Deploy admin UI for WeChat credentials
4. Test in WeChat sandbox environment
5. Enable in production

**Rollback:** Disable `wechat_enabled` setting, remove WeChat routes.

### Phase 4: ePay Integration (Week 7-8)
1. Obtain ePay official API documentation
2. Implement ePay provider based on documentation
3. Complete signature verification and webhook handling
4. Test in ePay test environment
5. Enable in production

**Rollback:** Disable `epay_enabled` setting.

### Phase 5: Renewal Reminders (Week 9)
1. Implement renewal reminder scheduled job
2. Deploy email templates for renewal reminders
3. Test with sample licenses expiring in controlled timeframes
4. Enable renewal reminders via setting toggle

**Rollback:** Set `renewal_reminders_enabled=false`.

### Deployment Checklist
- [ ] Database migrations applied (backward-compatible)
- [ ] Feature flags configured (`{provider}_enabled=false` initially)
- [ ] Admin can access Payment settings tab
- [ ] Test connection works for each provider
- [ ] Webhook URLs displayed correctly in admin UI
- [ ] Existing Stripe integration still functions
- [ ] Monitoring alerts configured for webhook failures

## Open Questions

**Q1:** Should we support WeChat Pay JSAPI and App modes in addition to Native (QR code)?

**Answer Deferred:** Start with Native mode only (most common for web checkouts). JSAPI requires OpenID which adds complexity. Can be added later if customer demand exists.

**Q2:** What is the exact ePay signature algorithm?

**Answer Deferred:** Requires official ePay API documentation. Implement as configurable (MD5/HMAC-SHA256) with override capability once confirmed.

**Q3:** Should renewal reminders include automatic payment retry links?

**Answer Deferred:** Current design uses manual renewal checkout links. Automatic retry would require storing payment methods (PCI compliance burden). Revisit if customer feedback indicates friction.

**Q4:** Should we support mixed-currency subscriptions (e.g., start in USD, renew in CNY)?

**Answer Deferred:** Not in initial implementation. Renewal should use same currency as original purchase. Can be added as upgrade path later.
