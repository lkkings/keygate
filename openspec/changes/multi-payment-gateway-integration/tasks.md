# Multi-Payment Gateway Integration - Tasks

## 1. Database Schema and Migrations

- [x] 1.1 Create migration for `payment_transactions` table with columns: id, provider, provider_transaction_id, license_id, plan_id, amount, currency, status, payment_method, metadata (JSONB), created_at, updated_at, and unique constraint on (provider, provider_transaction_id). Verify migration runs without errors on test database.

- [x] 1.2 Create migration to add multi-currency price columns to `plans` table: price_cny, price_hkd, stripe_price_id_cny, stripe_price_id_hkd (all nullable). Verify migration runs and existing plans remain unchanged.

- [x] 1.3 Create migration to add `payment_currency` TEXT and `renewal_for_license_id` TEXT columns to `licenses` table (both nullable). Verify migration applies and existing licenses work.

- [x] 1.4 Create `renewal_reminders` table with columns: id, license_id, reminder_type (30d/14d/7d/1d), sent_at, created_at, and unique constraint on (license_id, reminder_type). Verify table creation.

- [x] 1.5 Add indexes: idx_payment_transactions_license_id, idx_payment_transactions_provider_status, idx_licenses_valid_until, idx_renewal_reminders_license_type. Verify indexes exist via `\d+ table_name`.

## 2. Payment Provider Interface and Registry

- [x] 2.1 Create `internal/payment/provider.go` defining `PaymentProvider` interface with methods: CreateCheckout, VerifyWebhook, RefundPayment, GetPaymentStatus. Verify code compiles. (Note: FulfillPayment removed, now handled by PaymentService)

- [x] 2.2 Define shared types in `internal/payment/types.go`: CheckoutSession, CheckoutCustomer, WebhookEvent, PaymentStatus, RefundResult. Verify types compile and include required fields from design.md.

- [x] 2.3 Implement provider registry in `internal/payment/registry.go` with functions: RegisterProvider, GetProvider, ListEnabledProviders. Verify unit test can register and retrieve mock provider.

- [x] 2.4 Create `internal/payment/errors.go` with typed errors: ErrProviderNotFound, ErrProviderDisabled, ErrInvalidSignature, ErrPaymentNotFound. Verify errors implement error interface.

## 3. Stripe Provider Adapter

- [x] 3.1 Create `internal/payment/stripe/adapter.go` implementing PaymentProvider interface that wraps existing StripeHandler. Verify code compiles.

- [x] 3.2 Implement StripeProvider.CreateCheckout by delegating to existing CreateCheckoutSession. Verify existing Stripe checkout flow still works end-to-end.

- [x] 3.3 Implement StripeProvider.VerifyWebhook by delegating to existing webhook verification. Verify test webhook signature validation passes.

- [x] 3.4 StripeProvider fulfillment delegated to PaymentService. Verify test payment creates license correctly. (Architecture changed)

- [x] 3.5 Implement StripeProvider.RefundPayment and GetPaymentStatus. Verify unit tests for both methods pass.

- [x] 3.6 Register StripeProvider in init() function. Verify GetProvider("stripe") returns the adapter without errors.

## 4. Alipay Provider Implementation

- [x] 4.1 Create `internal/payment/alipay/client.go` with AlipayClient struct and methods: UnifiedOrder, VerifySign, RefundOrder, QueryOrder. Verify client instantiation with test credentials.

- [x] 4.2 Implement RSA2 signature generation in `internal/payment/alipay/signature.go` following Alipay specifications (sort params, SHA256WithRSA). Verify against Alipay test vectors.

- [x] 4.3 Implement RSA2 signature verification using Alipay public key. Verify using sample signed data from Alipay sandbox.

- [x] 4.4 Create `internal/payment/alipay/provider.go` implementing PaymentProvider interface. Verify code compiles and satisfies interface.

- [x] 4.5 Implement AlipayProvider.CreateCheckout calling alipay.trade.page.pay API with plan price_cny. Verify returns redirect URL in test mode.

- [x] 4.6 Implement AlipayProvider.VerifyWebhook parsing form parameters and verifying signature. Verify test webhook with valid signature passes, invalid fails.

- [x] 4.7 AlipayProvider fulfillment delegated to PaymentService. Verify test webhook fulfills license. (Architecture changed)

- [x] 4.8 Implement AlipayProvider.RefundPayment calling alipay.trade.refund API. Verify test refund request succeeds in sandbox.

- [x] 4.9 Implement AlipayProvider.GetPaymentStatus calling alipay.trade.query API. Verify returns correct status for test order.

- [x] 4.10 Register AlipayProvider in init() function. Verify GetProvider("alipay") returns provider when alipay_enabled=true.

## 5. WeChat Pay Provider Implementation

- [x] 5.1 Create `internal/payment/wechat/client.go` with WechatClient struct and methods: UnifiedOrder, VerifySign, RefundOrder, QueryOrder. Verify client instantiation.

- [x] 5.2 Implement MD5 signature generation in `internal/payment/wechat/signature.go` (sort params, append key, MD5 uppercase). Verify against WeChat test cases.

- [x] 5.3 Implement XML parsing and generation utilities in `internal/payment/wechat/xml.go` for WeChat API format. Verify round-trip parse and generate.

- [x] 5.4 Create `internal/payment/wechat/provider.go` implementing PaymentProvider interface. Verify code compiles.

- [x] 5.5 Implement WechatProvider.CreateCheckout calling unified order API with trade_type=NATIVE and plan price_cny (in cents). Verify returns code_url.

- [x] 5.6 Implement WechatProvider.VerifyWebhook parsing XML and verifying signature. Verify test webhook XML with valid signature passes.

- [x] 5.7 Implement WechatProvider.FulfillPayment handling result_code=SUCCESS and creating payment_transaction. Verify returns correct XML response.

- [x] 5.8 Implement WechatProvider.RefundPayment calling refund API with certificate authentication. Verify test refund (requires test certificate setup).

- [x] 5.9 Implement WechatProvider.GetPaymentStatus calling order query API. Verify returns correct trade_state.

- [x] 5.10 Register WechatProvider in init() function. Verify GetProvider("wechat") returns provider when wechat_enabled=true.

## 6. ePay Provider Implementation

- [x] 6.1 Create `internal/payment/epay/client.go` with EpayClient struct skeleton and method stubs. Verify code compiles with TODO comments.

- [x] 6.2 Create `internal/payment/epay/signature.go` with configurable signature algorithm (MD5/HMAC-SHA256). Verify both algorithms work with test data.

- [x] 6.3 Create `internal/payment/epay/provider.go` implementing PaymentProvider interface with TODO markers for API-specific details. Verify code compiles.

- [x] 6.4 Implement EpayProvider.CreateCheckout stub that returns NotImplemented error. Verify error is returned.

- [x] 6.5 Implement EpayProvider.VerifyWebhook stub with signature verification skeleton. Verify fails gracefully with clear error.

- [x] 6.6 Implement EpayProvider.FulfillPayment, RefundPayment, GetPaymentStatus stubs. Verify all return NotImplemented errors.

- [x] 6.7 Register EpayProvider in init() with epay_enabled=false default. Verify provider is registered but disabled.

## 7. Settings Management

- [x] 7.1 Add Alipay settings keys to `internal/model/settings.go`: alipay_enabled, alipay_app_id, alipay_private_key, alipay_public_key, alipay_sandbox_mode. Verify settings can be saved and retrieved.

- [x] 7.2 Add WeChat settings keys: wechat_enabled, wechat_app_id, wechat_merchant_id, wechat_api_key, wechat_payment_type, wechat_sandbox_mode. Verify settings persistence.

- [x] 7.3 Add ePay settings keys: epay_enabled, epay_merchant_id, epay_api_key, epay_api_secret, epay_test_mode. Verify settings work.

- [x] 7.4 Add renewal reminder settings: renewal_reminders_enabled, renewal_reminder_days. Verify default values load correctly.

- [x] 7.5 Implement settings validation functions for each provider (format checks, required fields). Verify validation errors are returned for invalid input.

## 8. Payment API Routes

- [x] 8.1 Create `internal/handler/payment/handler.go` with PaymentHandler struct containing provider registry. Verify handler instantiation.

- [x] 8.2 Implement POST `/api/v1/payment/:provider/checkout` route calling provider.CreateCheckout. Verify routes to correct provider and returns checkout session.

- [x] 8.3 Implement POST `/api/v1/payment/:provider/webhook` route calling provider.VerifyWebhook and PaymentService.HandleWebhook. Verify handles webhook correctly. (FulfillPayment now in service layer)

- [x] 8.4 Implement GET `/api/v1/payment/:provider/return` route for synchronous callbacks (Alipay). Verify redirects to success page with signature verification.

- [x] 8.5 Implement GET `/api/v1/payment/providers` route returning list of enabled providers with capabilities. Verify filters by plan's available currencies.

- [x] 8.6 Implement POST `/api/v1/payment/:provider/refund` admin route calling provider.RefundPayment. Verify requires admin auth and processes refund.

- [x] 8.7 Implement GET `/api/v1/payment/:provider/status/:transaction_id` route calling provider.GetPaymentStatus. Verify returns current status.

- [x] 8.8 Register all routes in `cmd/server/main.go`. Verify all routes are accessible and return correct status codes.

## 9. Admin Settings UI - Backend

- [x] 9.1 Create GET `/api/v1/admin/settings/payment` endpoint returning all payment provider settings (secrets masked). Verify returns structured provider config.

- [x] 9.2 Create PUT `/api/v1/admin/settings/payment/:provider` endpoint to update provider settings. Verify validates and saves settings, preserving existing secrets if not provided.

- [x] 9.3 Create POST `/api/v1/admin/settings/payment/:provider/test` endpoint calling provider test connection. Verify makes test API call and returns success/failure.

- [x] 9.4 Add webhook URL generation logic returning {base_url}/api/v1/payment/{provider}/webhook. Verify URLs are correct for all providers.

## 10. Admin Settings UI - Frontend

- [x] 10.1 Create `frontend/src/pages/admin/PaymentSettings.tsx` component with tab layout and provider cards. Verify page renders with correct structure.

- [x] 10.2 Implement StripeConfigCard component with fields for secret_key, publishable_key, webhook_secret, and enabled toggle. Verify form submission saves settings.

- [x] 10.3 Implement AlipayConfigCard component with fields for app_id, private_key (textarea), public_key (textarea), enabled and sandbox toggles. Verify saves and masks secrets.

- [x] 10.4 Implement WechatConfigCard component with fields for app_id, merchant_id, api_key, payment_type dropdown, enabled and sandbox toggles. Verify form works.

- [x] 10.5 Implement EpayConfigCard component with fields for merchant_id, api_key, api_secret, enabled and test_mode toggles. Verify renders correctly.

- [x] 10.6 Add webhook URL display with copy-to-clipboard button for each provider. Verify clicking copy button copies URL and shows confirmation.

- [x] 10.7 Implement "Test Connection" button for each provider calling test endpoint. Verify shows loading state, success toast, or error message.

- [x] 10.8 Add client-side validation for app_id formats (Alipay 16 digits, WeChat starts with "wx"). Verify validation errors display inline.

- [x] 10.9 Implement password field toggle (show/hide) for secret inputs. Verify eye icon toggles visibility correctly.

- [x] 10.10 Add responsive grid layout (2 columns desktop, 1 column mobile). Verify cards stack correctly on narrow viewports.

## 11. Multi-Currency Pricing UI

- [x] 11.1 Update `frontend/src/pages/admin/PlanForm.tsx` to add currency price fields: price_usd, price_cny, price_hkd. Verify fields render with currency symbols.

- [x] 11.2 Implement decimal-to-cents conversion on form submit and cents-to-decimal on load. Verify 99.00 displays as 99.00 and saves as 9900.

- [x] 11.3 Add validation ensuring prices are positive or null. Verify error displays for negative/zero values.

- [x] 11.4 Update plan list display to show "Multi-currency" badge if plan has >1 currency configured. Verify badge appears correctly.

- [x] 11.5 Update plan detail API response to include prices object: {usd: 9900, cny: 9900, hkd: null}. Verify API returns correct structure.

## 12. Checkout Flow - Frontend

- [x] 12.1 Create `frontend/src/components/PaymentMethodSelector.tsx` component displaying available payment providers. Verify fetches from /api/v1/payment/providers.

- [x] 12.2 Implement provider filtering based on plan's available currencies (hide Alipay if no price_cny). Verify Alipay only shows when CNY price exists.

- [x] 12.3 Add payment provider logos and labels (Stripe, 支付宝, 微信支付, ePay). Verify logos render correctly.

- [x] 12.4 Implement provider selection state and "Continue to Payment" button. Verify button is disabled until provider selected.

- [x] 12.5 On provider selection, call POST /api/v1/payment/{provider}/checkout and handle redirect or QR code display. Verify Stripe redirects, Alipay shows QR code.

- [x] 12.6 Implement QR code display component for Native payment modes (Alipay, WeChat). Verify QR code renders from code_url.

- [x] 12.7 Add payment pending status polling for QR code payments. Verify polls status API and updates UI when payment completes.

## 13. License Fulfillment Logic

- [x] 13.1 Implement provider-agnostic FulfillPayment in PaymentService. Verify creates license from payment event.

- [x] 13.2 Implement payment_transactions record creation in fulfillment flow. Verify transaction record is saved with correct provider, amount, currency.

- [x] 13.3 Implement idempotency check using (provider, provider_transaction_id) uniqueness. Verify duplicate webhook doesn't create duplicate license.

- [x] 13.4 Update license creation to store payment_currency field. Verify licenses created with CNY payments have currency="CNY".

- [x] 13.5 Implement renewal payment handling that extends existing license instead of creating new one. Verify renewal_for_license_id links correctly.

- [x] 13.6 Add audit logging for all fulfillment events (payment, renewal, refund). Verify logs include provider, transaction_id, action.

## 14. Renewal Reminder System

- [x] 14.1 Create `internal/service/renewal/reminder.go` with RenewalReminderService struct. Verify service instantiates correctly.

- [x] 14.2 Implement FindLicensesExpiringIn(days int) query excluding perpetual licenses and Stripe subscriptions. Verify query returns correct licenses.

- [x] 14.3 Implement wasReminderSent(licenseID, days) check against renewal_reminders table. Verify prevents duplicate reminders.

- [x] 14.4 Implement markReminderSent(licenseID, days) inserting to renewal_reminders. Verify record is created with correct timestamp.

- [x] 14.5 Create email template `templates/emails/renewal_reminder.html` with variables {{days_remaining}}, {{renewal_url}}, {{plan_name}}. Verify template renders correctly.

- [x] 14.6 Implement sendReminderEmail function calling email service with template. Verify email is sent with correct subject and content.

- [x] 14.7 Implement SendDueReminders() method looping through [30, 14, 7, 1] day intervals. Verify sends reminders for licenses at each interval.

- [x] 14.8 Create cron job registration for daily reminder check at 00:00 UTC. Verify job runs on schedule (test with shorter interval).

- [x] 14.9 Implement batch processing (100 licenses per batch) to avoid overwhelming email queue. Verify processes in batches correctly.

- [x] 14.10 Add renewal_reminders_enabled setting check before processing. Verify job exits early when disabled.

## 15. Renewal Checkout Flow

- [x] 15.1 Create GET `/renewal?license={id}` or `/renewal?token={signed}` route loading renewal page. Verify validates license ownership.

- [x] 15.2 Pre-fill email and pre-select current plan in renewal checkout form. Verify customer info is populated from license.

- [x] 15.3 Add "Upgrade Plan" option during renewal allowing selection of higher tier. Verify upgrade changes selected plan.

- [x] 15.4 Include is_renewal=true and original_license_id in checkout metadata. Verify metadata is passed to payment provider.

- [x] 15.5 Implement renewal payment webhook handler that extends valid_until instead of creating new license. Verify extension calculation is correct.

- [x] 15.6 Clear renewal_reminders records for license on successful renewal. Verify reminders reset for new period.

## 16. Refund Processing

- [x] 16.1 Add "Refund" button to license detail page in admin UI (admin only). Verify button appears and requires confirmation.

- [x] 16.2 Implement refund confirmation modal with amount input (full/partial). Verify modal shows transaction details.

- [x] 16.3 On refund confirm, call POST /api/v1/payment/{provider}/refund with transaction_id and amount. Verify API call succeeds.

- [x] 16.4 Update payment_transactions status to "refunded" and add refund_amount, refunded_at fields. Verify transaction record updates.

- [x] 16.5 Update license status based on refund policy (revoke immediately or at expiration). Verify license status changes correctly.

- [x] 16.6 Send refund confirmation email to customer. Verify email is sent with refund details.

## 17. Transaction History and Reporting

- [x] 17.1 Create GET `/api/v1/admin/transactions` endpoint returning paginated payment_transactions. Verify returns transactions with filtering options.

- [x] 17.2 Add filters for provider, status, date range to transaction list endpoint. Verify filtering works correctly.

- [x] 17.3 Create transaction list UI in admin panel showing provider, amount, currency, status, date. Verify table displays correctly.

- [x] 17.4 Add revenue report grouping by currency (separate totals per currency). Verify displays "Total: $X USD, ¥Y CNY, $Z HKD".

- [x] 17.5 Implement CSV export for transactions with all fields. Verify export downloads correct data.

## 18. Testing and Verification

- [x] 18.1 Write unit tests for all provider signature verification functions. Verify tests cover valid and invalid signatures.

- [x] 18.2 Write integration tests for complete checkout flow (Stripe, Alipay webhook simulation). Verify end-to-end license creation.

- [x] 18.3 Write unit tests for provider registry (register, get, list). Verify registry operations work correctly.

- [x] 18.4 Write unit tests for fulfillment service idempotency. Verify duplicate events don't create duplicate licenses.

- [x] 18.5 Write unit tests for renewal reminder scheduling logic. Verify correct licenses are selected at each interval.

- [x] 18.6 Test Alipay integration in sandbox environment (create checkout, complete payment, verify webhook). Verify full flow works.

- [x] 18.7 Test WeChat Pay integration in sandbox (create QR code, simulate payment callback). Verify QR code and webhook handling.

- [x] 18.8 Test multi-currency pricing (create plan with CNY price, checkout with Alipay). Verify correct currency is used.

- [x] 18.9 Test renewal flow (expire license, receive reminder, complete renewal). Verify license is extended correctly.

- [x] 18.10 Test refund flow for each provider (create payment, process refund). Verify refund succeeds and status updates.

## 19. Documentation

- [x] 19.1 Write admin guide for configuring payment providers (obtaining credentials, webhook setup). Verify guide is complete and accurate.

- [x] 19.2 Document webhook URLs and signature verification requirements per provider. Verify documentation matches implementation.

- [x] 19.3 Create troubleshooting guide for common payment issues (signature mismatch, missing currency price). Verify covers common errors.

- [x] 19.4 Document renewal reminder system (configuration, email templates, testing). Verify explains customization options.

- [x] 19.5 Add API documentation for new payment endpoints (OpenAPI/Swagger). Verify API docs are generated and accessible.

## 20. Deployment and Rollout

- [x] 20.1 Run database migrations in production during maintenance window. Verify migrations complete without errors and rollback plan is ready.

- [x] 20.2 Deploy backend with all providers initially disabled (feature flags). Verify existing Stripe integration still works.

- [x] 20.3 Deploy frontend with Payment settings tab. Verify admin can access settings and UI renders correctly.

- [x] 20.4 Enable Alipay in production after sandbox testing. Verify test checkout completes successfully.

- [x] 20.5 Enable WeChat Pay in production after sandbox testing. Verify QR code payment flow works.

- [x] 20.6 Enable renewal reminders with initial test group (10 licenses). Verify reminders are sent correctly.

- [x] 20.7 Monitor webhook failure rates and signature verification errors for 48 hours. Verify error rates are acceptable (<0.1%).

- [x] 20.8 Gradually enable all providers for all users. Verify no increase in error rates or support tickets.

- [x] 20.9 Document rollback procedure for each component. Verify rollback can be executed within 5 minutes.

- [x] 20.10 Conduct post-deployment review and update runbook. Verify all issues encountered are documented.
