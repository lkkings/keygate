# Alipay Integration Specification

## Purpose

Enables payment processing through Alipay (支付宝) with support for web payments and QR code payments, including signature verification for webhooks and return URLs.

## ADDED Requirements

### Requirement: Alipay Checkout Creation

The system SHALL create Alipay payment sessions supporting both web payment (redirect) and QR code (native scan) methods.

#### Scenario: Create web payment checkout
- **WHEN** customer initiates checkout with Alipay provider for a plan
- **THEN** system SHALL call Alipay's `alipay.trade.page.pay` API and return a redirect URL

#### Scenario: Create QR code payment checkout
- **WHEN** customer requests QR code payment method
- **THEN** system SHALL call Alipay's `alipay.trade.precreate` API and return QR code data

#### Scenario: Include plan metadata in checkout
- **WHEN** creating Alipay checkout
- **THEN** the request SHALL include `out_trade_no` (unique order ID), `total_amount`, `subject` (plan name), and `body` (plan description)

#### Scenario: Handle CNY currency
- **WHEN** plan has `price_cny` configured
- **THEN** checkout amount SHALL be in CNY (元) with 2 decimal places

#### Scenario: Plan missing CNY price
- **WHEN** checkout is requested with Alipay but plan has no `price_cny`
- **THEN** system SHALL return 400 Bad Request with error "plan does not support CNY payments"

### Requirement: Alipay Return URL Handling

The system SHALL process synchronous payment confirmation when Alipay redirects customers back to the return URL.

#### Scenario: Successful payment return
- **WHEN** Alipay redirects to `/api/v1/payment/alipay/return` with signed parameters
- **THEN** system SHALL verify signature, extract `out_trade_no` and `trade_status`, and redirect to success page

#### Scenario: Payment still processing
- **WHEN** return URL indicates `trade_status=WAIT_BUYER_PAY`
- **THEN** system SHALL redirect to pending status page

#### Scenario: Invalid signature on return
- **WHEN** return URL parameters fail signature verification
- **THEN** system SHALL log the failure and redirect to error page with "payment verification failed"

### Requirement: Alipay Webhook (Notify URL) Processing

The system SHALL receive and verify asynchronous payment notifications from Alipay.

#### Scenario: Payment success notification
- **WHEN** Alipay posts to `/api/v1/payment/alipay/notify` with `trade_status=TRADE_SUCCESS` or `TRADE_FINISHED`
- **THEN** system SHALL verify signature, fulfill the license, and respond with "success"

#### Scenario: Signature verification
- **WHEN** Alipay webhook is received
- **THEN** system SHALL verify RSA2 signature using Alipay's public key from settings

#### Scenario: Invalid signature on webhook
- **WHEN** webhook signature verification fails
- **THEN** system SHALL return 400 Bad Request and NOT fulfill the payment

#### Scenario: Duplicate notification
- **WHEN** webhook is received for an already-fulfilled `out_trade_no`
- **THEN** system SHALL return "success" without processing again (idempotent)

#### Scenario: Webhook acknowledgment
- **WHEN** payment is successfully fulfilled
- **THEN** system MUST respond with exactly "success" (Alipay requires this exact string)

### Requirement: Alipay Refund Processing

The system SHALL support refunding Alipay payments via the `alipay.trade.refund` API.

#### Scenario: Full refund
- **WHEN** admin initiates refund for an Alipay payment
- **THEN** system SHALL call `alipay.trade.refund` with `out_trade_no` and original amount

#### Scenario: Partial refund
- **WHEN** admin initiates partial refund
- **THEN** system SHALL call `alipay.trade.refund` with `refund_amount` less than original amount

#### Scenario: Refund already processed
- **WHEN** refund API returns `ACQ.TRADE_HAS_SUCCESS` (already refunded)
- **THEN** system SHALL treat it as success (idempotent)

#### Scenario: Refund failure
- **WHEN** Alipay refund API returns error
- **THEN** system SHALL log the error code and message, and return failure to admin

### Requirement: Alipay Payment Status Query

The system SHALL query Alipay payment status using the `alipay.trade.query` API.

#### Scenario: Query payment by out_trade_no
- **WHEN** system needs to verify payment status
- **THEN** it SHALL call `alipay.trade.query` with `out_trade_no`

#### Scenario: Payment not found
- **WHEN** query returns `ACQ.TRADE_NOT_EXIST`
- **THEN** system SHALL return status "not_found"

#### Scenario: Payment completed
- **WHEN** query returns `trade_status=TRADE_SUCCESS`
- **THEN** system SHALL return status "completed"

### Requirement: Alipay Signature Generation

The system SHALL generate RSA2 signatures for all outbound Alipay API requests using the merchant's private key.

#### Scenario: Sign API request
- **WHEN** making API call to Alipay
- **THEN** system SHALL sort parameters alphabetically, concatenate key=value pairs with &, and sign with SHA256WithRSA

#### Scenario: Include charset and sign_type
- **WHEN** signing request
- **THEN** parameters MUST include `charset=utf-8` and `sign_type=RSA2`

### Requirement: Alipay Configuration Validation

The system SHALL validate Alipay credentials before saving or enabling the provider.

#### Scenario: Validate app_id format
- **WHEN** admin enters Alipay app_id
- **THEN** system SHALL validate it is a numeric string of 16 digits

#### Scenario: Validate private key format
- **WHEN** admin enters private key
- **THEN** system SHALL validate it is PEM-formatted RSA private key

#### Scenario: Validate public key format
- **WHEN** admin enters Alipay public key
- **THEN** system SHALL validate it is PEM-formatted RSA public key

#### Scenario: Test credentials with API call
- **WHEN** admin clicks "Test Connection"
- **THEN** system SHALL make a test API call (e.g., `alipay.system.oauth.token` with invalid code) and verify signature verification works

### Requirement: Transaction Reference Storage

The system SHALL store Alipay-specific transaction data in the `payment_transactions` table and `licenses.payment_reference`.

#### Scenario: Store out_trade_no
- **WHEN** Alipay payment is created
- **THEN** system SHALL store the generated `out_trade_no` in `licenses.payment_reference`

#### Scenario: Store trade_no
- **WHEN** Alipay webhook provides `trade_no` (Alipay's internal transaction ID)
- **THEN** system SHALL store it in `payment_transactions.provider_transaction_id`

#### Scenario: Store payment metadata
- **WHEN** payment is fulfilled
- **THEN** system SHALL store buyer_id, buyer_logon_id, trade_status in `payment_transactions.metadata` as JSONB
