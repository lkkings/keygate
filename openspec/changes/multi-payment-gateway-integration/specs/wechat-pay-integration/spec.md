# WeChat Pay Integration Specification

## Purpose

Enables payment processing through WeChat Pay (微信支付) with support for JSAPI (公众号/小程序), Native (扫码), and App payment modes, including signature verification and payment callbacks.

## ADDED Requirements

### Requirement: WeChat Pay Checkout Creation

The system SHALL create WeChat Pay unified orders supporting JSAPI, Native, and App payment modes.

#### Scenario: Create Native payment (QR code)
- **WHEN** customer initiates checkout with WeChat Pay Native mode
- **THEN** system SHALL call WeChat's unified order API with `trade_type=NATIVE` and return `code_url` for QR code generation

#### Scenario: Create JSAPI payment (in-app)
- **WHEN** customer initiates checkout with WeChat Pay JSAPI mode
- **THEN** system SHALL call unified order API with `trade_type=JSAPI` and user's `openid`, returning `prepay_id` for JSAPI SDK

#### Scenario: Create App payment
- **WHEN** customer initiates checkout with WeChat Pay App mode
- **THEN** system SHALL call unified order API with `trade_type=APP` and return signed payment parameters for mobile SDK

#### Scenario: Include required parameters
- **WHEN** creating WeChat Pay order
- **THEN** request SHALL include `appid`, `mch_id`, `nonce_str`, `body` (plan name), `out_trade_no`, `total_fee` (in 分/cents), `spbill_create_ip`, `notify_url`, `trade_type`

#### Scenario: Amount in CNY cents
- **WHEN** plan has `price_cny` of 99.00 CNY
- **THEN** WeChat Pay order SHALL specify `total_fee=9900` (in 分)

#### Scenario: Plan missing CNY price
- **WHEN** checkout is requested with WeChat Pay but plan has no `price_cny`
- **THEN** system SHALL return 400 Bad Request with error "plan does not support CNY payments"

### Requirement: WeChat Pay Signature Generation

The system SHALL generate MD5 signatures for all outbound WeChat Pay API requests using the merchant API key.

#### Scenario: Sign unified order request
- **WHEN** calling WeChat Pay unified order API
- **THEN** system SHALL sort parameters alphabetically, concatenate key=value pairs with &, append `&key={api_key}`, and compute MD5 hash in uppercase

#### Scenario: Exclude sign field from signature
- **WHEN** generating signature
- **THEN** the `sign` parameter itself MUST be excluded from the signature calculation

#### Scenario: Handle empty values
- **WHEN** parameter value is empty string
- **THEN** that parameter MUST be excluded from signature calculation

### Requirement: WeChat Pay Webhook Processing

The system SHALL receive and verify payment result notifications from WeChat Pay.

#### Scenario: Payment success notification
- **WHEN** WeChat posts XML to `/api/v1/payment/wechat/webhook` with `result_code=SUCCESS`
- **THEN** system SHALL verify signature, fulfill the license, and respond with XML containing `<return_code>SUCCESS</return_code>`

#### Scenario: Signature verification
- **WHEN** WeChat webhook is received
- **THEN** system SHALL verify the `sign` field by recalculating signature with the API key

#### Scenario: Invalid signature on webhook
- **WHEN** webhook signature verification fails
- **THEN** system SHALL return XML with `<return_code>FAIL</return_code>` and NOT fulfill payment

#### Scenario: Duplicate notification
- **WHEN** webhook is received for already-fulfilled `out_trade_no`
- **THEN** system SHALL return SUCCESS XML without processing again (idempotent)

#### Scenario: Payment failure notification
- **WHEN** webhook has `result_code=FAIL`
- **THEN** system SHALL log the failure reason and return SUCCESS acknowledgment (do not retry)

#### Scenario: XML response format
- **WHEN** responding to WeChat webhook
- **THEN** response MUST be XML with `<xml><return_code>SUCCESS</return_code><return_msg>OK</return_msg></xml>`

### Requirement: WeChat Pay Refund Processing

The system SHALL support refunding WeChat Pay payments via the refund API.

#### Scenario: Full refund
- **WHEN** admin initiates refund for WeChat Pay payment
- **THEN** system SHALL call WeChat refund API with `out_trade_no`, `out_refund_no` (unique refund ID), `total_fee`, and `refund_fee`

#### Scenario: Partial refund
- **WHEN** admin initiates partial refund
- **THEN** `refund_fee` SHALL be less than `total_fee`

#### Scenario: Refund requires certificate
- **WHEN** calling WeChat refund API
- **THEN** system SHALL use merchant certificate (`apiclient_cert.p12` or equivalent) for mutual TLS authentication

#### Scenario: Refund already processed
- **WHEN** refund API returns error indicating already refunded
- **THEN** system SHALL treat it as success (idempotent)

### Requirement: WeChat Pay Order Query

The system SHALL query WeChat Pay order status using the order query API.

#### Scenario: Query order by out_trade_no
- **WHEN** system needs to verify payment status
- **THEN** it SHALL call order query API with `out_trade_no`

#### Scenario: Order not found
- **WHEN** query returns `err_code=ORDERNOTEXIST`
- **THEN** system SHALL return status "not_found"

#### Scenario: Order paid
- **WHEN** query returns `trade_state=SUCCESS`
- **THEN** system SHALL return status "completed" and fulfill if not already fulfilled

#### Scenario: Order unpaid
- **WHEN** query returns `trade_state=NOTPAY`
- **THEN** system SHALL return status "pending"

### Requirement: WeChat Pay Payment Mode Selection

The system SHALL allow admin to configure which WeChat Pay payment mode to use.

#### Scenario: Configure payment mode in settings
- **WHEN** admin configures WeChat Pay
- **THEN** settings SHALL include `wechat_payment_type` with values: `NATIVE`, `JSAPI`, or `APP`

#### Scenario: Default to Native mode
- **WHEN** WeChat Pay is enabled but `wechat_payment_type` is not set
- **THEN** system SHALL default to `NATIVE` (QR code) mode

#### Scenario: JSAPI requires openid
- **WHEN** payment mode is JSAPI
- **THEN** checkout request MUST include customer's WeChat `openid` parameter

#### Scenario: Missing openid for JSAPI
- **WHEN** JSAPI payment is requested without `openid`
- **THEN** system SHALL return 400 Bad Request with error "openid required for JSAPI payment"

### Requirement: WeChat Pay Configuration Validation

The system SHALL validate WeChat Pay credentials before saving or enabling the provider.

#### Scenario: Validate app_id format
- **WHEN** admin enters WeChat app_id
- **THEN** system SHALL validate it starts with "wx" and is 18 characters long

#### Scenario: Validate mch_id format
- **WHEN** admin enters merchant ID
- **THEN** system SHALL validate it is a numeric string

#### Scenario: Validate api_key format
- **WHEN** admin enters API key
- **THEN** system SHALL validate it is 32 characters long

#### Scenario: Test credentials with API call
- **WHEN** admin clicks "Test Connection"
- **THEN** system SHALL make a test unified order API call and verify signature validation works

### Requirement: Transaction Reference Storage

The system SHALL store WeChat Pay-specific transaction data in the database.

#### Scenario: Store out_trade_no
- **WHEN** WeChat Pay order is created
- **THEN** system SHALL store the generated `out_trade_no` in `licenses.payment_reference`

#### Scenario: Store transaction_id
- **WHEN** WeChat webhook provides `transaction_id` (WeChat's internal ID)
- **THEN** system SHALL store it in `payment_transactions.provider_transaction_id`

#### Scenario: Store payment metadata
- **WHEN** payment is fulfilled
- **THEN** system SHALL store `openid`, `trade_type`, `bank_type`, `time_end` in `payment_transactions.metadata` as JSONB

### Requirement: WeChat Pay Sandbox Mode

The system SHALL support WeChat Pay sandbox environment for testing.

#### Scenario: Sandbox API endpoint
- **WHEN** `wechat_sandbox_mode` setting is true
- **THEN** system SHALL use sandbox API endpoint instead of production

#### Scenario: Sandbox signature key
- **WHEN** in sandbox mode
- **THEN** system SHALL obtain sandbox signature key via `sandboxnew/pay/getsignkey` API before making test orders
