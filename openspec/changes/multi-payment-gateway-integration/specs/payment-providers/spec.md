# Payment Providers Specification

## Purpose

Defines a unified interface for integrating multiple payment gateways (Stripe, Alipay, WeChat Pay, ePay) with consistent behavior across checkout, webhooks, refunds, and payment verification.

## ADDED Requirements

### Requirement: PaymentProvider Interface

The system SHALL define a `PaymentProvider` interface that all payment gateway implementations MUST implement, supporting checkout creation, webhook verification, payment fulfillment, refund processing, and status queries.

#### Scenario: Provider implements all required methods
- **WHEN** a new payment provider is registered
- **THEN** it MUST implement CreateCheckout, VerifyWebhook, FulfillPayment, RefundPayment, and GetPaymentStatus methods

#### Scenario: Provider returns standardized response format
- **WHEN** any provider method is called
- **THEN** the response MUST follow a standardized format with success/error status, provider-specific transaction ID, and metadata

### Requirement: Provider Registration

The system SHALL maintain a provider registry that allows dynamic registration and lookup of payment providers by name.

#### Scenario: Register a new provider
- **WHEN** system initializes with payment providers enabled in settings
- **THEN** each enabled provider MUST be registered in the provider registry

#### Scenario: Lookup provider by name
- **WHEN** a payment request specifies provider name (stripe|alipay|wechat|epay)
- **THEN** the registry MUST return the corresponding provider implementation or error if not found

#### Scenario: List available providers
- **WHEN** frontend requests available payment methods
- **THEN** the system SHALL return only providers that are enabled in settings and have valid credentials

### Requirement: Provider-Agnostic Payment Routes

The system SHALL expose REST API endpoints that route requests to the appropriate provider based on URL parameter.

#### Scenario: Create checkout with provider routing
- **WHEN** client sends POST to `/api/v1/payment/:provider/checkout`
- **THEN** the request MUST be routed to the provider specified in the URL

#### Scenario: Invalid provider name
- **WHEN** client requests `/api/v1/payment/invalid-provider/checkout`
- **THEN** the system SHALL return 400 Bad Request with error message "unsupported payment provider"

### Requirement: Provider Configuration

The system SHALL store provider-specific configuration in the settings table with keys prefixed by provider name.

#### Scenario: Alipay configuration keys
- **WHEN** admin configures Alipay
- **THEN** settings SHALL include keys: `alipay_enabled`, `alipay_app_id`, `alipay_private_key`, `alipay_public_key`

#### Scenario: WeChat Pay configuration keys
- **WHEN** admin configures WeChat Pay
- **THEN** settings SHALL include keys: `wechat_enabled`, `wechat_app_id`, `wechat_merchant_id`, `wechat_api_key`, `wechat_payment_type`

#### Scenario: ePay configuration keys
- **WHEN** admin configures ePay
- **THEN** settings SHALL include keys: `epay_enabled`, `epay_merchant_id`, `epay_api_key`, `epay_api_secret`

#### Scenario: Provider disabled
- **WHEN** a provider's `{provider}_enabled` setting is false
- **THEN** checkout requests for that provider SHALL return 503 Service Unavailable with message "payment provider not available"

### Requirement: Webhook URL Generation

The system SHALL provide webhook URLs for each provider following the pattern `/api/v1/payment/:provider/webhook`.

#### Scenario: Display webhook URLs in admin UI
- **WHEN** admin views payment settings
- **THEN** the UI SHALL display the webhook URL for each provider as `{base_url}/api/v1/payment/{provider}/webhook`

#### Scenario: Webhook endpoint routing
- **WHEN** a webhook is received at `/api/v1/payment/alipay/webhook`
- **THEN** it MUST be handled by the Alipay provider's VerifyWebhook method

### Requirement: Error Handling and Logging

The system SHALL log all payment provider interactions with sufficient detail for debugging while redacting sensitive data.

#### Scenario: Log successful payment
- **WHEN** a payment is successfully processed
- **THEN** system SHALL log provider name, transaction ID, amount, currency, and license ID (but NOT full payment credentials or signatures)

#### Scenario: Log payment failure
- **WHEN** a payment fails
- **THEN** system SHALL log provider name, error code, error message, and request context

#### Scenario: Provider timeout
- **WHEN** a provider API call times out
- **THEN** system SHALL return 503 Service Unavailable and log the timeout with provider name and operation

### Requirement: Provider Testing

The system SHALL provide a test connection endpoint for each configured provider.

#### Scenario: Test Alipay connection
- **WHEN** admin clicks "Test Connection" for Alipay
- **THEN** system SHALL make a minimal API call to Alipay (e.g., query merchant info) and return success/failure

#### Scenario: Test with invalid credentials
- **WHEN** test connection is triggered with invalid API keys
- **THEN** system SHALL return specific error message from the provider (e.g., "invalid app_id" or "signature verification failed")
