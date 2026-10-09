# Subscription Renewal Reminders Specification

## Purpose

Provides automated email reminders for subscription renewals when using payment providers (Alipay, WeChat Pay, ePay) that do not support native recurring billing, ensuring customers are notified to manually renew before their license expires.

## ADDED Requirements

### Requirement: Renewal Reminder Schedule

The system SHALL send renewal reminder emails at predefined intervals before license expiration for non-recurring payment methods.

#### Scenario: 30-day advance reminder
- **WHEN** license paid via Alipay/WeChat/ePay is 30 days from expiration
- **THEN** system SHALL send renewal reminder email with subject "Your {product_name} license expires in 30 days"

#### Scenario: 14-day advance reminder
- **WHEN** license paid via Alipay/WeChat/ePay is 14 days from expiration
- **THEN** system SHALL send renewal reminder email with subject "Your {product_name} license expires in 2 weeks"

#### Scenario: 7-day advance reminder
- **WHEN** license paid via Alipay/WeChat/ePay is 7 days from expiration
- **THEN** system SHALL send renewal reminder email with subject "Your {product_name} license expires in 1 week"

#### Scenario: 1-day advance reminder
- **WHEN** license paid via Alipay/WeChat/ePay is 1 day from expiration
- **THEN** system SHALL send renewal reminder email with subject "Your {product_name} license expires tomorrow"

#### Scenario: Stripe subscriptions excluded
- **WHEN** license is paid via Stripe with active subscription
- **THEN** system SHALL NOT send manual renewal reminders (Stripe handles auto-renewal)

#### Scenario: Perpetual licenses excluded
- **WHEN** license type is perpetual
- **THEN** system SHALL NOT send renewal reminders (no expiration)

### Requirement: Renewal Reminder Deduplication

The system SHALL ensure each reminder is sent only once per license per renewal cycle.

#### Scenario: Track sent reminders
- **WHEN** renewal reminder is sent
- **THEN** system SHALL record reminder type (30d, 14d, 7d, 1d) and timestamp in database

#### Scenario: Skip already-sent reminder
- **WHEN** reminder job runs and finds reminder already sent for this license at this interval
- **THEN** system SHALL NOT resend the same reminder

#### Scenario: Reset on renewal
- **WHEN** license is renewed (new payment received)
- **THEN** system SHALL clear reminder tracking so new reminders can be sent for the new period

#### Scenario: Prevent duplicate sends within 24 hours
- **WHEN** reminder was sent less than 24 hours ago
- **THEN** system SHALL NOT send another reminder at the same interval (idempotency guard)

### Requirement: Renewal Reminder Email Content

The system SHALL include renewal instructions and payment links in reminder emails.

#### Scenario: Include renewal checkout link
- **WHEN** generating renewal reminder email
- **THEN** email SHALL include clickable link to renewal checkout page with license pre-populated

#### Scenario: Display expiration date
- **WHEN** generating renewal reminder email
- **THEN** email SHALL clearly state the exact expiration date in user's timezone (from settings)

#### Scenario: Show plan name and price
- **WHEN** generating renewal reminder email
- **THEN** email SHALL include plan name and current price in the currency originally paid

#### Scenario: Multiple payment method options
- **WHEN** generating renewal reminder email
- **THEN** email SHALL list available payment methods (Alipay, WeChat Pay, ePay, Stripe) the customer can use

#### Scenario: Email template customization
- **WHEN** admin configures renewal reminder email template
- **THEN** system SHALL support variables: `{{product_name}}`, `{{plan_name}}`, `{{expiration_date}}`, `{{days_remaining}}`, `{{renewal_url}}`, `{{price}}`

### Requirement: Renewal Checkout Flow

The system SHALL provide a renewal-specific checkout flow that links new payment to existing license.

#### Scenario: Renewal link includes license ID
- **WHEN** customer clicks renewal link from email
- **THEN** URL SHALL include license ID or encrypted token: `/renew?license={license_id}` or `/renew?token={signed_token}`

#### Scenario: Pre-fill customer information
- **WHEN** customer lands on renewal page
- **THEN** form SHALL pre-fill email address from existing license

#### Scenario: Same plan pre-selected
- **WHEN** renewal checkout loads
- **THEN** system SHALL pre-select the same plan the customer currently has

#### Scenario: Allow plan upgrade during renewal
- **WHEN** customer is on renewal page
- **THEN** system SHALL offer option to upgrade to higher-tier plan

#### Scenario: Extend existing license on payment
- **WHEN** renewal payment is completed
- **THEN** system SHALL extend `valid_until` date of existing license (not create new license)

#### Scenario: Link renewal payment to license
- **WHEN** processing renewal payment
- **THEN** `payment_transactions` SHALL include `renewal_for_license_id` field linking to original license

### Requirement: Renewal Payment Metadata

The system SHALL mark renewal payments with metadata to distinguish from new purchases.

#### Scenario: Set renewal flag in checkout metadata
- **WHEN** creating payment session for renewal
- **THEN** system SHALL include `is_renewal=true` in payment metadata

#### Scenario: Include original license ID
- **WHEN** creating renewal payment session
- **THEN** metadata SHALL include `original_license_id` field

#### Scenario: Webhook identifies renewal
- **WHEN** payment webhook is received for renewal
- **THEN** system SHALL detect `is_renewal=true` and extend existing license instead of creating new one

#### Scenario: Renewal audit log
- **WHEN** renewal payment is processed
- **THEN** audit log SHALL record action as "renewed" with old and new expiration dates

### Requirement: Renewal Reminder Scheduler

The system SHALL run a periodic job to check for licenses requiring renewal reminders.

#### Scenario: Daily reminder check job
- **WHEN** renewal reminder job runs (recommended: daily at 00:00 UTC)
- **THEN** system SHALL query licenses expiring in 30, 14, 7, and 1 days

#### Scenario: Batch processing
- **WHEN** renewal job finds multiple licenses to remind
- **THEN** system SHALL process them in batches of 100 to avoid overwhelming email queue

#### Scenario: Job runs only for active licenses
- **WHEN** renewal job queries licenses
- **THEN** it SHALL only include licenses with status "active" (exclude suspended, revoked, expired)

#### Scenario: Timezone-aware scheduling
- **WHEN** calculating reminder dates
- **THEN** system SHALL use license's associated timezone from settings (or default to UTC)

### Requirement: Admin Renewal Reminder Configuration

The system SHALL allow admin to configure renewal reminder behavior.

#### Scenario: Enable/disable renewal reminders
- **WHEN** admin configures email settings
- **THEN** system SHALL provide toggle for `renewal_reminders_enabled`

#### Scenario: Customize reminder days
- **WHEN** admin wants different reminder schedule
- **THEN** system SHALL accept comma-separated values for `renewal_reminder_days` (e.g., "30,14,7,3,1")

#### Scenario: Disable specific reminder intervals
- **WHEN** admin sets `renewal_reminder_days` to "30,7"
- **THEN** system SHALL only send reminders at 30 and 7 days (skip 14 and 1)

#### Scenario: Per-provider reminder toggle
- **WHEN** admin configures payment providers
- **THEN** each provider SHALL have `{provider}_renewal_reminders_enabled` setting to selectively enable/disable

### Requirement: Grace Period Handling

The system SHALL define grace period behavior for expired licenses pending renewal.

#### Scenario: Grace period duration
- **WHEN** license expires without renewal
- **THEN** system SHALL grant 7-day grace period where license remains functional

#### Scenario: Grace period notification
- **WHEN** license is in grace period (expired but within 7 days)
- **THEN** system SHALL send daily reminder with urgent tone

#### Scenario: Grace period expiration
- **WHEN** grace period ends without renewal
- **THEN** license SHALL transition to "expired" status and become non-functional

#### Scenario: Renewal during grace period
- **WHEN** payment is received during grace period
- **THEN** license SHALL be reactivated and `valid_until` extended from original expiration date (not from payment date)

### Requirement: Renewal Analytics

The system SHALL track renewal metrics for reporting.

#### Scenario: Track renewal rate
- **WHEN** generating reports
- **THEN** system SHALL calculate renewal rate as (renewed licenses / licenses due for renewal) per period

#### Scenario: Track reminders sent
- **WHEN** generating email analytics
- **THEN** system SHALL report count of renewal reminders sent per interval (30d, 14d, 7d, 1d)

#### Scenario: Track reminder conversion
- **WHEN** generating reports
- **THEN** system SHALL show which reminder interval led to renewal (if customer renewed within 24 hours of receiving reminder)

#### Scenario: Non-renewal reporting
- **WHEN** generating churn reports
- **THEN** system SHALL identify licenses that expired without renewal attempt
