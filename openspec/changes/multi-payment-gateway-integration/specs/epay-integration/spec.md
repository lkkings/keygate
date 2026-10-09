# ePay Integration Specification

## Purpose

Enables payment processing through ePay (易派支付) with support for web-based payments, signature verification, and callback handling.

## ADDED Requirements

### Requirement: ePay Checkout Creation

The system SHALL create ePay payment sessions for web-based checkout.

#### Scenario: Create ePay payment checkout
- **WHEN** customer initiates checkout with ePay provider for a plan
- **THEN** system SHALL call ePay's payment API and return a redirect URL or payment form

#### Scenario: Include order parameters
- **WHEN** creating ePay checkout
- **THEN** the request SHALL include merchant_id, order_id (unique), amount, currency, product_name, notify_url, return_url

#### Scenario: Support multiple currencies
- **WHEN** plan has price in supported currency (CNY, USD, HKD)
- **THEN** checkout SHALL use the appropriate currency code

#### Scenario: Plan missing supported currency price
- **WHEN** checkout is requested with ePay but plan has no price in ePay-supported currencies
- **THEN** system SHALL return 400 Bad Request with error "plan does not support currencies accepted by ePay"

### Requirement: ePay Signature Generation

The system SHALL generate signatures for all outbound ePay API requests using the merchant's API secret.

#### Scenario: Sign payment request
- **WHEN** calling ePay payment API
- **THEN** system SHALL generate signature according to ePay's signing rules (typically MD5 or HMAC-SHA256 based on documentation)

#### Scenario: Include timestamp and nonce
- **WHEN** signing request
- **THEN** parameters MUST include timestamp and random nonce for replay protection

#### Scenario: Sort parameters for signing
- **WHEN** generating signature
- **THEN** system SHALL sort parameters alphabetically by key before concatenation

### Requirement: ePay Return URL Handling

The system SHALL process synchronous payment confirmation when ePay redirects customers back to the return URL.

#### Scenario: Successful payment return
- **WHEN** ePay redirects to `/api/v1/payment/epay/return` with signed parameters
- **THEN** system SHALL verify signature, extract order_id and payment status, and redirect to success page

#### Scenario: Payment pending return
- **WHEN** return URL indicates payment is still processing
- **THEN** system SHALL redirect to pending status page

#### Scenario: Invalid signature on return
- **WHEN** return URL parameters fail signature verification
- **THEN** system SHALL log the failure and redirect to error page with "payment verification failed"

### Requirement: ePay Webhook (Notify URL) Processing

The system SHALL receive and verify asynchronous payment notifications from ePay.

#### Scenario: Payment success notification
- **WHEN** ePay posts to `/api/v1/payment/epay/webhook` with successful payment status
- **THEN** system SHALL verify signature, fulfill the license, and respond with acknowledgment (format depends on ePay spec)

#### Scenario: Signature verification
- **WHEN** ePay webhook is received
- **THEN** system SHALL verify signature using ePay's API secret from settings

#### Scenario: Invalid signature on webhook
- **WHEN** webhook signature verification fails
- **THEN** system SHALL return 400 Bad Request and NOT fulfill the payment

#### Scenario: Duplicate notification
- **WHEN** webhook is received for an already-fulfilled order_id
- **THEN** system SHALL return success acknowledgment without processing again (idempotent)

#### Scenario: Webhook acknowledgment format
- **WHEN** payment is successfully fulfilled
- **THEN** system SHALL respond in the format required by ePay (e.g., "SUCCESS" or JSON with success status)

### Requirement: ePay Refund Processing

The system SHALL support refunding ePay payments via the refund API.

#### Scenario: Full refund
- **WHEN** admin initiates refund for an ePay payment
- **THEN** system SHALL call ePay refund API with order_id and original amount

#### Scenario: Partial refund support
- **WHEN** ePay supports partial refunds
- **THEN** system SHALL allow specifying refund_amount less than original amount

#### Scenario: Refund not supported
- **WHEN** ePay does not support programmatic refunds
- **THEN** system SHALL return error message "manual refund required, contact ePay support" and log the refund request for manual processing

#### Scenario: Refund already processed
- **WHEN** refund API indicates order already refunded
- **THEN** system SHALL treat it as success (idempotent)

### Requirement: ePay Payment Status Query

The system SHALL query ePay payment status when available via their API.

#### Scenario: Query payment by order_id
- **WHEN** system needs to verify payment status
- **THEN** it SHALL call ePay's order query API with order_id

#### Scenario: Payment not found
- **WHEN** query returns order not found error
- **THEN** system SHALL return status "not_found"

#### Scenario: Payment completed
- **WHEN** query returns successful payment status
- **THEN** system SHALL return status "completed"

#### Scenario: Payment pending
- **WHEN** query returns pending status
- **THEN** system SHALL return status "pending"

### Requirement: ePay Configuration Validation

The system SHALL validate ePay credentials before saving or enabling the provider.

#### Scenario: Validate merchant_id format
- **WHEN** admin enters ePay merchant_id
- **THEN** system SHALL validate it matches expected format (depends on ePay specification)

#### Scenario: Validate API key format
- **WHEN** admin enters API key
- **THEN** system SHALL validate it is non-empty and meets minimum length requirement

#### Scenario: Validate API secret format
- **WHEN** admin enters API secret
- **THEN** system SHALL validate it is non-empty and meets minimum length requirement

#### Scenario: Test credentials with API call
- **WHEN** admin clicks "Test Connection"
- **THEN** system SHALL make a test API call to ePay (e.g., query merchant info or create test order) and verify credentials are valid

### Requirement: Transaction Reference Storage

The system SHALL store ePay-specific transaction data in the database.

#### Scenario: Store order_id
- **WHEN** ePay payment is created
- **THEN** system SHALL store the generated order_id in `licenses.payment_reference`

#### Scenario: Store ePay transaction_id
- **WHEN** ePay webhook provides their internal transaction_id
- **THEN** system SHALL store it in `payment_transactions.provider_transaction_id`

#### Scenario: Store payment metadata
- **WHEN** payment is fulfilled
- **THEN** system SHALL store payment_method, transaction_time, and any other relevant fields from ePay callback in `payment_transactions.metadata` as JSONB

### Requirement: ePay Sandbox/Test Mode

The system SHALL support ePay sandbox or test environment if available.

#### Scenario: Test mode configuration
- **WHEN** `epay_test_mode` setting is true
- **THEN** system SHALL use ePay's test API endpoints instead of production

#### Scenario: Test mode indicators in admin UI
- **WHEN** ePay is configured in test mode
- **THEN** admin UI SHALL display badge indicating "Test Mode" next to ePay settings

### Requirement: ePay Error Handling

The system SHALL handle ePay-specific error codes and provide meaningful error messages.

#### Scenario: Insufficient merchant balance
- **WHEN** ePay returns error indicating merchant account issue
- **THEN** system SHALL log the error and return "payment temporarily unavailable"

#### Scenario: Network timeout to ePay
- **WHEN** ePay API call times out
- **THEN** system SHALL retry once, and if still failing, return 503 Service Unavailable

#### Scenario: ePay maintenance window
- **WHEN** ePay returns maintenance error
- **THEN** system SHALL return user-friendly message "ePay is under maintenance, please try again later"

### Requirement: ePay Documentation Placeholder

The implementation SHALL reference ePay's official API documentation for exact specifications.

#### Scenario: Document API endpoint URLs
- **WHEN** implementing ePay provider
- **THEN** code comments SHALL include links to ePay's official API documentation

#### Scenario: Document signature algorithm
- **WHEN** implementing signature generation
- **THEN** code SHALL document the exact signature algorithm used by ePay (MD5, HMAC-SHA256, etc.)

#### Scenario: Incomplete specification handling
- **WHEN** ePay specification details are unavailable during initial implementation
- **THEN** code SHALL include TODO comments marking areas requiring ePay documentation
