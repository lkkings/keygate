# Database Migrations for Multi-Payment Gateway Integration

## Task 19.4 & 19.5: Database Schema Changes

This document outlines the database migrations required for the multi-payment gateway integration.

---

## Migration Overview

The following tables and schema changes are required:

1. **payment_transactions** - Store all payment records
2. **renewal_reminders** - Track sent reminder emails
3. **settings** - Store payment provider configurations
4. **licenses** - Add payment-related fields

---

## Migration 001: Create payment_transactions Table

```sql
CREATE TABLE IF NOT EXISTS payment_transactions (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    provider_name VARCHAR(50) NOT NULL,
    provider_tx_id VARCHAR(255) NOT NULL,
    session_id VARCHAR(255),
    license_id BIGINT,
    amount BIGINT NOT NULL COMMENT 'Amount in smallest currency unit (cents)',
    currency VARCHAR(3) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    payment_method VARCHAR(50),
    customer_email VARCHAR(255),
    refund_amount BIGINT DEFAULT 0,
    refunded_at DATETIME,
    metadata TEXT COMMENT 'JSON string',
    processed_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    UNIQUE KEY unique_provider_tx (provider_name, provider_tx_id),
    INDEX idx_license_id (license_id),
    INDEX idx_status (status),
    INDEX idx_created_at (created_at),
    INDEX idx_customer_email (customer_email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

**Purpose**: Store all payment transactions from all providers with idempotency guarantee via unique constraint on `(provider_name, provider_tx_id)`.

---

## Migration 002: Create renewal_reminders Table

```sql
CREATE TABLE IF NOT EXISTS renewal_reminders (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    license_id VARCHAR(255) NOT NULL,
    days_before INT NOT NULL COMMENT 'Days before expiration (30, 14, 7, 1)',
    sent_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE KEY unique_reminder (license_id, days_before),
    INDEX idx_license_id (license_id),
    INDEX idx_sent_at (sent_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

**Purpose**: Track which renewal reminders have been sent to prevent duplicates. The unique constraint ensures each reminder is sent only once per interval.

---

## Migration 003: Add Multi-Currency Fields to Plans

```sql
ALTER TABLE plans 
ADD COLUMN IF NOT EXISTS price_cny BIGINT COMMENT 'Price in Chinese Yuan (fen)' AFTER price_usd,
ADD COLUMN IF NOT EXISTS price_hkd BIGINT COMMENT 'Price in Hong Kong Dollars (cents)' AFTER price_cny,
ADD COLUMN IF NOT EXISTS renewal_days INT DEFAULT 365 COMMENT 'Days to extend license on renewal' AFTER updates_days;
```

**Purpose**: Support multi-currency pricing for different payment providers.

---

## Migration 004: Update Licenses Table for Revocation

```sql
ALTER TABLE licenses
ADD COLUMN IF NOT EXISTS revoked_at DATETIME COMMENT 'When license was revoked (e.g., after refund)' AFTER canceled_at;
```

**Purpose**: Track license revocations separately from cancellations.

---

## Migration 005: Add Payment Provider Settings

```sql
-- Settings table should already exist, but ensure these keys are supported

-- Example settings for Stripe
INSERT INTO settings (key, value) VALUES 
    ('stripe_enabled', 'false'),
    ('stripe_secret_key', ''),
    ('stripe_publishable_key', ''),
    ('stripe_webhook_secret', '')
ON DUPLICATE KEY UPDATE key=key;

-- Example settings for Alipay
INSERT INTO settings (key, value) VALUES
    ('alipay_enabled', 'false'),
    ('alipay_app_id', ''),
    ('alipay_private_key', ''),
    ('alipay_public_key', ''),
    ('alipay_gateway', 'https://openapi.alipay.com/gateway.do')
ON DUPLICATE KEY UPDATE key=key;

-- Example settings for WeChat Pay
INSERT INTO settings (key, value) VALUES
    ('wechat_enabled', 'false'),
    ('wechat_app_id', ''),
    ('wechat_mch_id', ''),
    ('wechat_api_key', ''),
    ('wechat_api_cert', '')
ON DUPLICATE KEY UPDATE key=key;

-- Example settings for ePay
INSERT INTO settings (key, value) VALUES
    ('epay_enabled', 'false'),
    ('epay_merchant_id', ''),
    ('epay_api_key', ''),
    ('epay_api_secret', ''),
    ('epay_gateway', '')
ON DUPLICATE KEY UPDATE key=key;

-- Renewal reminder settings
INSERT INTO settings (key, value) VALUES
    ('renewal_reminders_enabled', 'true')
ON DUPLICATE KEY UPDATE key=key;
```

---

## Running Migrations

### Using Go Migrate

```bash
# Install migrate tool
go install -tags 'mysql' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run all pending migrations
migrate -path db/migrations -database "mysql://user:pass@tcp(localhost:3306)/keygate" up

# Rollback last migration
migrate -path db/migrations -database "mysql://user:pass@tcp(localhost:3306)/keygate" down 1

# Check migration status
migrate -path db/migrations -database "mysql://user:pass@tcp(localhost:3306)/keygate" version
```

### Manual Execution

```bash
# Connect to database
mysql -u keygate_user -p keygate_db

# Run each migration in order
source db/migrations/001_payment_transactions.sql
source db/migrations/002_renewal_reminders.sql
source db/migrations/003_multi_currency_plans.sql
source db/migrations/004_license_revocation.sql
source db/migrations/005_payment_settings.sql
```

---

## Rollback Scripts

### Rollback 005: Remove Payment Settings

```sql
DELETE FROM settings WHERE key IN (
    'stripe_enabled', 'stripe_secret_key', 'stripe_publishable_key', 'stripe_webhook_secret',
    'alipay_enabled', 'alipay_app_id', 'alipay_private_key', 'alipay_public_key', 'alipay_gateway',
    'wechat_enabled', 'wechat_app_id', 'wechat_mch_id', 'wechat_api_key', 'wechat_api_cert',
    'epay_enabled', 'epay_merchant_id', 'epay_api_key', 'epay_api_secret', 'epay_gateway',
    'renewal_reminders_enabled'
);
```

### Rollback 004: Remove Revocation Field

```sql
ALTER TABLE licenses DROP COLUMN IF EXISTS revoked_at;
```

### Rollback 003: Remove Multi-Currency Fields

```sql
ALTER TABLE plans 
DROP COLUMN IF EXISTS renewal_days,
DROP COLUMN IF EXISTS price_hkd,
DROP COLUMN IF EXISTS price_cny;
```

### Rollback 002: Drop renewal_reminders Table

```sql
DROP TABLE IF EXISTS renewal_reminders;
```

### Rollback 001: Drop payment_transactions Table

```sql
DROP TABLE IF EXISTS payment_transactions;
```

---

## Data Migration Notes

### Existing Licenses

If you have existing licenses, no data migration is needed. The new fields are nullable or have defaults.

### Existing Plans

For existing plans, you may want to set CNY and HKD prices:

```sql
-- Example: Set CNY price based on USD price (exchange rate ~7)
UPDATE plans 
SET price_cny = FLOOR(price_usd * 7) 
WHERE price_usd > 0 AND price_cny IS NULL;

-- Example: Set HKD price based on USD price (exchange rate ~7.8)
UPDATE plans 
SET price_hkd = FLOOR(price_usd * 7.8) 
WHERE price_usd > 0 AND price_hkd IS NULL;
```

---

## Verification

After running migrations, verify the schema:

```sql
-- Check payment_transactions table
DESCRIBE payment_transactions;
SHOW INDEX FROM payment_transactions;

-- Check renewal_reminders table
DESCRIBE renewal_reminders;
SHOW INDEX FROM renewal_reminders;

-- Check plans table has new columns
SHOW COLUMNS FROM plans LIKE 'price_%';
SHOW COLUMNS FROM plans LIKE 'renewal_days';

-- Check licenses table has revoked_at
SHOW COLUMNS FROM licenses LIKE 'revoked_at';

-- Verify settings
SELECT key, value FROM settings WHERE key LIKE '%_enabled';
```

---

## Production Deployment

### Pre-Deployment Checklist

1. **Backup database**:
   ```bash
   mysqldump -u root -p keygate_db > backup_$(date +%Y%m%d_%H%M%S).sql
   ```

2. **Test migrations on staging**:
   - Run migrations on staging database
   - Verify application works
   - Test rollback procedure

3. **Plan maintenance window**:
   - Migrations should be fast (<1 second each)
   - Consider running during low-traffic period
   - Prepare rollback plan

### Deployment Steps

1. Put application in maintenance mode (optional)
2. Run migrations
3. Deploy new application code
4. Verify critical functions:
   - Payment checkout works
   - Webhooks process correctly
   - Admin panel loads
5. Monitor error logs
6. Remove maintenance mode

### Post-Deployment Verification

```sql
-- Verify unique constraints work
INSERT INTO payment_transactions (provider_name, provider_tx_id, amount, currency, status)
VALUES ('test', 'txn_123', 1000, 'USD', 'completed');

-- This should fail with duplicate key error
INSERT INTO payment_transactions (provider_name, provider_tx_id, amount, currency, status)
VALUES ('test', 'txn_123', 1000, 'USD', 'completed');

-- Clean up test data
DELETE FROM payment_transactions WHERE provider_name = 'test';
```

---

## Troubleshooting

### Migration Fails

**Check for:**
- Existing tables with same name
- Insufficient privileges
- Character set conflicts
- Foreign key constraints

**Solution:**
```sql
-- Check existing tables
SHOW TABLES LIKE 'payment_%';

-- Check user privileges
SHOW GRANTS FOR CURRENT_USER();

-- Check character set
SHOW VARIABLES LIKE 'character_set%';
```

### Duplicate Key Error During Migration

If unique constraint already exists:
```sql
-- Drop existing constraint
ALTER TABLE payment_transactions DROP INDEX unique_provider_tx;

-- Recreate with new definition
ALTER TABLE payment_transactions ADD UNIQUE KEY unique_provider_tx (provider_name, provider_tx_id);
```

---

## Support

For migration issues:
1. Check migration logs
2. Verify database user privileges
3. Test on local/staging environment first
4. Contact support with error messages and migration version
