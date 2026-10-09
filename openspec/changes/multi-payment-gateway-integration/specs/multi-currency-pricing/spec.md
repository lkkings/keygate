# Multi-Currency Pricing Specification

## Purpose

Enables plans to be priced in multiple currencies (USD, CNY, HKD, etc.) so customers can pay in their preferred currency through different payment providers.

## ADDED Requirements

### Requirement: Plan Multi-Currency Price Fields

The system SHALL extend the plans table to store prices in multiple currencies.

#### Scenario: Add CNY price field
- **WHEN** plan is created or updated
- **THEN** system SHALL accept `price_cny` field as integer (in 分/cents, e.g., 9900 = 99.00 CNY)

#### Scenario: Add HKD price field
- **WHEN** plan is created or updated
- **THEN** system SHALL accept `price_hkd` field as integer (in cents, e.g., 78800 = 788.00 HKD)

#### Scenario: Keep USD price in existing field
- **WHEN** plan has USD pricing
- **THEN** system SHALL continue using existing `price_usd` field (or base `price` field if that's current structure)

#### Scenario: Optional currency prices
- **WHEN** plan is saved
- **THEN** currency price fields SHALL be optional (nullable) - plan does not need prices in all currencies

#### Scenario: Zero is invalid
- **WHEN** currency price is provided
- **THEN** it MUST be greater than zero or null (zero is not a valid price)

### Requirement: Stripe Price ID per Currency

The system SHALL store Stripe price IDs for each currency separately.

#### Scenario: Store stripe_price_id_cny
- **WHEN** plan is configured with Stripe price for CNY
- **THEN** system SHALL store the Stripe price ID in `stripe_price_id_cny` field

#### Scenario: Store stripe_price_id_hkd
- **WHEN** plan is configured with Stripe price for HKD
- **THEN** system SHALL store the Stripe price ID in `stripe_price_id_hkd` field

#### Scenario: Keep existing stripe_price_id
- **WHEN** plan has USD Stripe pricing
- **THEN** system SHALL continue using existing `stripe_price_id` field

### Requirement: Admin UI Multi-Currency Input

The system SHALL provide admin UI for entering prices in multiple currencies.

#### Scenario: Display currency price fields in plan form
- **WHEN** admin creates or edits a plan
- **THEN** form SHALL display input fields for price_usd, price_cny, and price_hkd

#### Scenario: Currency labels and symbols
- **WHEN** displaying currency fields
- **THEN** each field SHALL be labeled with currency name and symbol (e.g., "Price (CNY) ¥", "Price (USD) $")

#### Scenario: Decimal input conversion
- **WHEN** admin enters "99.00" in CNY price field
- **THEN** system SHALL convert to 9900 cents before saving

#### Scenario: Display saved prices as decimals
- **WHEN** loading plan with price_cny = 9900
- **THEN** form SHALL display "99.00" in the CNY field

#### Scenario: Currency field validation
- **WHEN** admin enters invalid price (negative or non-numeric)
- **THEN** form SHALL show validation error "price must be a positive number"

### Requirement: Payment Provider Currency Selection

The system SHALL select the appropriate currency price based on the payment provider.

#### Scenario: Alipay uses CNY price
- **WHEN** customer checks out with Alipay
- **THEN** system SHALL use plan's `price_cny` field

#### Scenario: WeChat Pay uses CNY price
- **WHEN** customer checks out with WeChat Pay
- **THEN** system SHALL use plan's `price_cny` field

#### Scenario: ePay uses configured currency
- **WHEN** customer checks out with ePay
- **THEN** system SHALL use price in the first available currency ePay supports (CNY, USD, HKD in that priority)

#### Scenario: Stripe uses currency from frontend
- **WHEN** customer checks out with Stripe
- **THEN** frontend SHALL specify currency preference, and system uses corresponding `stripe_price_id_xxx`

#### Scenario: Missing currency price
- **WHEN** payment provider requires a currency not configured for the plan
- **THEN** checkout SHALL return 400 Bad Request with "plan does not support {currency} pricing"

### Requirement: Currency Display in Frontend

The system SHALL display available payment methods based on configured currency prices.

#### Scenario: Show Alipay only if CNY price exists
- **WHEN** displaying payment method options for a plan
- **THEN** Alipay SHALL appear only if `price_cny` is not null

#### Scenario: Show WeChat Pay only if CNY price exists
- **WHEN** displaying payment method options for a plan
- **THEN** WeChat Pay SHALL appear only if `price_cny` is not null

#### Scenario: Show multiple Stripe options
- **WHEN** plan has prices in multiple currencies
- **THEN** Stripe payment option SHALL allow customer to choose currency (if Stripe supports it in their account)

#### Scenario: Display price in selected currency
- **WHEN** customer selects a payment method
- **THEN** UI SHALL display the price in the currency that payment method will use

### Requirement: License Currency Recording

The system SHALL record which currency was used for a license purchase.

#### Scenario: Store payment currency in license
- **WHEN** license is created from payment
- **THEN** system SHALL store the currency code (CNY, USD, HKD) in `licenses` table or `payment_transactions.currency`

#### Scenario: Store payment amount in original currency
- **WHEN** recording transaction
- **THEN** `payment_transactions.amount` SHALL be in the original currency (not converted)

#### Scenario: Display purchase price in admin
- **WHEN** admin views license details
- **THEN** system SHALL display original purchase price with currency symbol (e.g., "¥99.00 CNY")

### Requirement: Currency Conversion for Reports

The system SHALL NOT perform automatic currency conversion in reports.

#### Scenario: Revenue reports show original currencies
- **WHEN** generating revenue reports
- **THEN** transactions SHALL be listed in their original currencies without conversion

#### Scenario: Currency-grouped totals
- **WHEN** displaying total revenue
- **THEN** system SHALL show separate totals per currency (e.g., "Total: $1,234 USD, ¥56,780 CNY, $4,321 HKD")

#### Scenario: No built-in exchange rates
- **WHEN** system displays multi-currency data
- **THEN** it SHALL NOT attempt automatic currency conversion (operator can use external tools if needed)

### Requirement: Migration from Single Currency

The system SHALL support migrating existing plans from single currency to multi-currency.

#### Scenario: Preserve existing USD prices
- **WHEN** multi-currency migration runs
- **THEN** existing `price` values SHALL be preserved in `price_usd` field

#### Scenario: Null for new currency fields
- **WHEN** multi-currency migration runs
- **THEN** new fields `price_cny` and `price_hkd` SHALL default to NULL

#### Scenario: Existing licenses unaffected
- **WHEN** multi-currency is deployed
- **THEN** existing licenses SHALL continue to work without requiring currency data updates

### Requirement: API Response Currency Format

The system SHALL include currency information in API responses.

#### Scenario: Plan API includes all currency prices
- **WHEN** API returns plan details
- **THEN** response SHALL include `prices` object with all configured currencies: `{"usd": 9900, "cny": 9900, "hkd": 78800}`

#### Scenario: Null currencies omitted
- **WHEN** plan has no price in a currency
- **THEN** that currency key SHALL be omitted from the `prices` object (or have null value)

#### Scenario: License API includes payment currency
- **WHEN** API returns license details
- **THEN** response SHALL include `payment_currency` and `payment_amount` fields
