# Payment Settings UI Specification

## Purpose

Provides admin interface for configuring payment provider credentials, testing connections, viewing webhook URLs, and enabling/disabling payment methods through a unified settings panel.

## ADDED Requirements

### Requirement: Payment Tab in Settings

The system SHALL add a dedicated Payment tab to the admin settings page.

#### Scenario: Display Payment tab
- **WHEN** admin navigates to Settings page
- **THEN** tabs SHALL include "General", "Email", "Payment", "Users" (Payment inserted before Users)

#### Scenario: Payment tab access control
- **WHEN** non-admin user views settings
- **THEN** Payment tab SHALL NOT be visible (admin/owner roles only)

#### Scenario: Tab navigation state
- **WHEN** admin clicks Payment tab
- **THEN** URL SHALL update to `/admin/settings?tab=payment` and state persists on refresh

### Requirement: Provider Configuration Cards

The system SHALL display separate configuration cards for each payment provider.

#### Scenario: Display Stripe card
- **WHEN** Payment tab is loaded
- **THEN** Stripe configuration card SHALL be displayed first with label "Stripe"

#### Scenario: Display Alipay card
- **WHEN** Payment tab is loaded
- **THEN** Alipay configuration card SHALL be displayed with label "支付宝 (Alipay)"

#### Scenario: Display WeChat Pay card
- **WHEN** Payment tab is loaded
- **THEN** WeChat Pay configuration card SHALL be displayed with label "微信支付 (WeChat Pay)"

#### Scenario: Display ePay card
- **WHEN** Payment tab is loaded
- **THEN** ePay configuration card SHALL be displayed with label "易派支付 (ePay)"

#### Scenario: Card visual state when disabled
- **WHEN** provider is disabled
- **THEN** card SHALL have reduced opacity and "Disabled" badge

#### Scenario: Card visual state when enabled
- **WHEN** provider is enabled
- **THEN** card SHALL have full opacity and "Enabled" badge in green

### Requirement: Stripe Configuration Form

The system SHALL provide form fields for Stripe configuration.

#### Scenario: Stripe secret key input
- **WHEN** displaying Stripe configuration
- **THEN** form SHALL include password-type input for `stripe_secret_key` with placeholder "sk_live_***" or "sk_test_***"

#### Scenario: Stripe publishable key input
- **WHEN** displaying Stripe configuration
- **THEN** form SHALL include text input for `stripe_publishable_key` with placeholder "pk_live_***"

#### Scenario: Stripe webhook secret input
- **WHEN** displaying Stripe configuration
- **THEN** form SHALL include password-type input for `stripe_webhook_secret`

#### Scenario: Stripe webhook URL display
- **WHEN** displaying Stripe configuration
- **THEN** form SHALL show read-only text field with value `{base_url}/api/v1/payment/stripe/webhook` and copy button

#### Scenario: Stripe enabled toggle
- **WHEN** displaying Stripe configuration
- **THEN** form SHALL include toggle switch for `stripe_enabled`

#### Scenario: Stripe test mode indicator
- **WHEN** Stripe keys start with "sk_test_" or "pk_test_"
- **THEN** card SHALL display "Test Mode" badge

### Requirement: Alipay Configuration Form

The system SHALL provide form fields for Alipay configuration.

#### Scenario: Alipay app_id input
- **WHEN** displaying Alipay configuration
- **THEN** form SHALL include text input for `alipay_app_id` with placeholder "2021001234567890"

#### Scenario: Alipay private key input
- **WHEN** displaying Alipay configuration
- **THEN** form SHALL include textarea for `alipay_private_key` with placeholder "-----BEGIN RSA PRIVATE KEY-----"

#### Scenario: Alipay public key input
- **WHEN** displaying Alipay configuration
- **THEN** form SHALL include textarea for `alipay_public_key` with placeholder "-----BEGIN PUBLIC KEY-----"

#### Scenario: Alipay return URL display
- **WHEN** displaying Alipay configuration
- **THEN** form SHALL show read-only field with value `{base_url}/api/v1/payment/alipay/return` and copy button

#### Scenario: Alipay notify URL display
- **WHEN** displaying Alipay configuration
- **THEN** form SHALL show read-only field with value `{base_url}/api/v1/payment/alipay/notify` and copy button

#### Scenario: Alipay enabled toggle
- **WHEN** displaying Alipay configuration
- **THEN** form SHALL include toggle switch for `alipay_enabled`

#### Scenario: Alipay sandbox mode toggle
- **WHEN** displaying Alipay configuration
- **THEN** form SHALL include optional toggle for `alipay_sandbox_mode`

### Requirement: WeChat Pay Configuration Form

The system SHALL provide form fields for WeChat Pay configuration.

#### Scenario: WeChat app_id input
- **WHEN** displaying WeChat Pay configuration
- **THEN** form SHALL include text input for `wechat_app_id` with placeholder "wx1234567890abcdef"

#### Scenario: WeChat merchant_id input
- **WHEN** displaying WeChat Pay configuration
- **THEN** form SHALL include text input for `wechat_merchant_id` with placeholder "1234567890"

#### Scenario: WeChat api_key input
- **WHEN** displaying WeChat Pay configuration
- **THEN** form SHALL include password-type input for `wechat_api_key` with 32-character max length

#### Scenario: WeChat payment type selector
- **WHEN** displaying WeChat Pay configuration
- **THEN** form SHALL include dropdown for `wechat_payment_type` with options: "Native (扫码)", "JSAPI (公众号/小程序)", "App (移动应用)"

#### Scenario: WeChat webhook URL display
- **WHEN** displaying WeChat Pay configuration
- **THEN** form SHALL show read-only field with value `{base_url}/api/v1/payment/wechat/webhook` and copy button

#### Scenario: WeChat enabled toggle
- **WHEN** displaying WeChat Pay configuration
- **THEN** form SHALL include toggle switch for `wechat_enabled`

#### Scenario: WeChat sandbox mode toggle
- **WHEN** displaying WeChat Pay configuration
- **THEN** form SHALL include optional toggle for `wechat_sandbox_mode`

### Requirement: ePay Configuration Form

The system SHALL provide form fields for ePay configuration.

#### Scenario: ePay merchant_id input
- **WHEN** displaying ePay configuration
- **THEN** form SHALL include text input for `epay_merchant_id`

#### Scenario: ePay api_key input
- **WHEN** displaying ePay configuration
- **THEN** form SHALL include password-type input for `epay_api_key`

#### Scenario: ePay api_secret input
- **WHEN** displaying ePay configuration
- **THEN** form SHALL include password-type input for `epay_api_secret`

#### Scenario: ePay webhook URL display
- **WHEN** displaying ePay configuration
- **THEN** form SHALL show read-only field with value `{base_url}/api/v1/payment/epay/webhook` and copy button

#### Scenario: ePay enabled toggle
- **WHEN** displaying ePay configuration
- **THEN** form SHALL include toggle switch for `epay_enabled`

#### Scenario: ePay test mode toggle
- **WHEN** displaying ePay configuration
- **THEN** form SHALL include optional toggle for `epay_test_mode`

### Requirement: Test Connection Functionality

The system SHALL provide "Test Connection" button for each provider to validate credentials.

#### Scenario: Test connection button placement
- **WHEN** displaying provider configuration card
- **THEN** card SHALL include "Test Connection" button below credentials

#### Scenario: Test connection loading state
- **WHEN** admin clicks "Test Connection"
- **THEN** button SHALL show loading spinner and text "Testing..."

#### Scenario: Test connection success
- **WHEN** test connection succeeds
- **THEN** UI SHALL display green checkmark toast notification with message "{Provider} connection successful"

#### Scenario: Test connection failure
- **WHEN** test connection fails
- **THEN** UI SHALL display red error toast with specific error message from API (e.g., "Invalid API key" or "Signature verification failed")

#### Scenario: Test connection disabled when fields empty
- **WHEN** required credential fields are empty
- **THEN** "Test Connection" button SHALL be disabled with tooltip "Please fill in all required fields"

#### Scenario: Test connection requires save first
- **WHEN** credentials are modified but not saved
- **THEN** "Test Connection" button SHALL be disabled with tooltip "Save settings before testing"

### Requirement: Save and Validation

The system SHALL validate and save payment provider settings.

#### Scenario: Save button state
- **WHEN** any setting is modified
- **THEN** "Save Settings" button SHALL become enabled

#### Scenario: Client-side validation
- **WHEN** admin attempts to save with invalid fields
- **THEN** form SHALL show inline validation errors without submitting

#### Scenario: Alipay app_id format validation
- **WHEN** Alipay app_id does not match 16-digit pattern
- **THEN** form SHALL show error "App ID must be 16 digits"

#### Scenario: WeChat app_id format validation
- **WHEN** WeChat app_id does not start with "wx" or is not 18 characters
- **THEN** form SHALL show error "App ID must start with 'wx' and be 18 characters"

#### Scenario: Private key format validation
- **WHEN** RSA private key does not contain "BEGIN RSA PRIVATE KEY" or "BEGIN PRIVATE KEY"
- **THEN** form SHALL show error "Invalid RSA private key format"

#### Scenario: Save success notification
- **WHEN** settings are successfully saved
- **THEN** UI SHALL display green toast "Payment settings saved successfully"

#### Scenario: Save failure notification
- **WHEN** save request fails
- **THEN** UI SHALL display red toast with error message from API

### Requirement: Webhook URL Copy Functionality

The system SHALL provide one-click copy for webhook URLs.

#### Scenario: Copy button displays
- **WHEN** webhook URL field is shown
- **THEN** a copy icon button SHALL appear next to the URL

#### Scenario: Copy to clipboard
- **WHEN** admin clicks copy button
- **THEN** URL SHALL be copied to clipboard and button shows checkmark briefly

#### Scenario: Copy success feedback
- **WHEN** URL is copied
- **THEN** tooltip SHALL appear with text "Copied!" for 2 seconds

### Requirement: Sensitive Data Handling

The system SHALL properly handle sensitive credential data in the UI.

#### Scenario: Password fields for secrets
- **WHEN** displaying API keys, secrets, or private keys
- **THEN** input type SHALL be "password" with toggle to show/hide

#### Scenario: Placeholder for saved secrets
- **WHEN** loading settings with existing secret
- **THEN** input SHALL show placeholder "•••••••••••••••" instead of actual value

#### Scenario: Clear on focus
- **WHEN** admin clicks into password field with placeholder
- **THEN** placeholder SHALL clear, allowing new value entry

#### Scenario: Empty means unchanged
- **WHEN** admin saves form with secret field empty/placeholder
- **THEN** backend SHALL keep existing secret (not delete it)

#### Scenario: Show/hide toggle
- **WHEN** admin clicks eye icon on password field
- **THEN** input type SHALL toggle between "password" and "text"

### Requirement: Provider Status Indicators

The system SHALL display real-time status indicators for each payment provider.

#### Scenario: Enabled status badge
- **WHEN** provider is enabled in settings
- **THEN** card SHALL display green "Enabled" badge in header

#### Scenario: Disabled status badge
- **WHEN** provider is disabled in settings
- **THEN** card SHALL display gray "Disabled" badge in header

#### Scenario: Configuration incomplete warning
- **WHEN** provider is enabled but missing required credentials
- **THEN** card SHALL display yellow "Configuration Incomplete" badge

#### Scenario: Test mode indicator
- **WHEN** provider is in sandbox/test mode
- **THEN** card SHALL display orange "Test Mode" badge

### Requirement: Responsive Layout

The system SHALL provide responsive layout for payment settings.

#### Scenario: Desktop layout
- **WHEN** viewport width >= 1024px
- **THEN** provider cards SHALL be displayed in 2-column grid

#### Scenario: Tablet layout
- **WHEN** viewport width between 768px and 1023px
- **THEN** provider cards SHALL be displayed in single column

#### Scenario: Mobile layout
- **WHEN** viewport width < 768px
- **THEN** provider cards SHALL stack vertically with full width

#### Scenario: Form field stacking
- **WHEN** viewport width < 768px
- **THEN** form labels and inputs SHALL stack vertically (not side-by-side)
