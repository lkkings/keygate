-- Keygate Database Schema - Complete initialization
-- This file consolidates all previous migrations into a single schema

-- =============================================================================
-- CORE TABLES
-- =============================================================================

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    name TEXT,
    role TEXT NOT NULL DEFAULT 'user' CHECK (role IN ('owner', 'admin', 'user')),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(LOWER(email));

-- Products table
CREATE TABLE IF NOT EXISTS products (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    license_type TEXT NOT NULL CHECK (license_type IN ('perpetual', 'subscription', 'saas')),
    download_url TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Plans table
CREATE TABLE IF NOT EXISTS plans (
    id TEXT PRIMARY KEY,
    product_id TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    slug TEXT NOT NULL,
    billing_interval TEXT NOT NULL CHECK (billing_interval IN ('monthly', 'yearly', 'perpetual', '')),
    price_usd INTEGER NOT NULL DEFAULT 0,
    price_cny INTEGER NOT NULL DEFAULT 0,
    price_hkd INTEGER NOT NULL DEFAULT 0,
    max_activations INTEGER NOT NULL DEFAULT 1,
    max_seats INTEGER NOT NULL DEFAULT 1,
    trial_days INTEGER NOT NULL DEFAULT 0,
    grace_days INTEGER NOT NULL DEFAULT 0,
    license_model TEXT NOT NULL DEFAULT 'standard' CHECK (license_model IN ('standard', 'floating')),
    stripe_price_id TEXT,
    stripe_renewal_price_id TEXT,
    checkout_id TEXT,
    features JSONB,
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_product_slug UNIQUE (product_id, slug)
);

CREATE INDEX IF NOT EXISTS idx_plans_product_id ON plans(product_id);
CREATE INDEX IF NOT EXISTS idx_plans_stripe_price_id ON plans(stripe_price_id);

-- Licenses table
CREATE TABLE IF NOT EXISTS licenses (
    id TEXT PRIMARY KEY,
    license_key TEXT UNIQUE NOT NULL,
    license_key_hash TEXT UNIQUE NOT NULL,
    user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    product_id TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    plan_id TEXT REFERENCES plans(id) ON DELETE SET NULL,
    email TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('active', 'suspended', 'expired', 'canceled', 'trial')),
    max_activations INTEGER NOT NULL DEFAULT 1,
    max_seats INTEGER NOT NULL DEFAULT 1,
    valid_from TIMESTAMP,
    valid_until TIMESTAMP,
    grace_until TIMESTAMP,
    trial_until TIMESTAMP,
    license_model TEXT NOT NULL DEFAULT 'standard' CHECK (license_model IN ('standard', 'floating')),

    -- Subscription fields
    subscription_id TEXT,
    subscription_status TEXT,
    subscription_current_period_end TIMESTAMP,
    subscription_cancel_at_period_end BOOLEAN DEFAULT FALSE,
    subscription_state_synced BOOLEAN DEFAULT TRUE,

    -- Payment fields
    payment_provider TEXT,
    payment_transaction_id TEXT,
    last_payment_at TIMESTAMP,
    next_billing_at TIMESTAMP,

    -- Stripe sync
    stripe_synced_at TIMESTAMP,

    -- Suspension tracking
    suspended_by TEXT,
    suspended_at TIMESTAMP,
    suspension_reason TEXT,

    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_licenses_email ON licenses(LOWER(email));
CREATE INDEX IF NOT EXISTS idx_licenses_user_id ON licenses(user_id);
CREATE INDEX IF NOT EXISTS idx_licenses_product_id ON licenses(product_id);
CREATE INDEX IF NOT EXISTS idx_licenses_plan_id ON licenses(plan_id);
CREATE INDEX IF NOT EXISTS idx_licenses_status ON licenses(status);
CREATE INDEX IF NOT EXISTS idx_licenses_subscription_id ON licenses(subscription_id);
CREATE INDEX IF NOT EXISTS idx_licenses_valid_until ON licenses(valid_until) WHERE valid_until IS NOT NULL;

-- Activations table
CREATE TABLE IF NOT EXISTS activations (
    id TEXT PRIMARY KEY,
    license_id TEXT NOT NULL REFERENCES licenses(id) ON DELETE CASCADE,
    device_name TEXT,
    device_fingerprint TEXT,
    ip_address TEXT,
    user_agent TEXT,
    metadata JSONB,
    activated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deactivated_at TIMESTAMP,
    lease_expires_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_activations_license_id ON activations(license_id);
CREATE INDEX IF NOT EXISTS idx_activations_device_fingerprint ON activations(device_fingerprint);
CREATE INDEX IF NOT EXISTS idx_activations_lease_expires_at ON activations(lease_expires_at) WHERE lease_expires_at IS NOT NULL;

-- =============================================================================
-- PAYMENT TABLES
-- =============================================================================

-- Payment transactions table
CREATE TABLE IF NOT EXISTS payment_transactions (
    id TEXT PRIMARY KEY,
    license_id TEXT REFERENCES licenses(id) ON DELETE SET NULL,
    provider_name TEXT NOT NULL,
    provider_tx_id TEXT NOT NULL,
    amount NUMERIC(10, 2) NOT NULL,
    currency TEXT NOT NULL DEFAULT 'USD',
    status TEXT NOT NULL CHECK (status IN ('pending', 'completed', 'failed', 'refunded')),
    customer_email TEXT,
    customer_name TEXT,
    refund_amount NUMERIC(10, 2) DEFAULT 0,
    refunded_at TIMESTAMP,
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_provider_tx UNIQUE (provider_name, provider_tx_id)
);

CREATE INDEX IF NOT EXISTS idx_payment_transactions_license_id ON payment_transactions(license_id);
CREATE INDEX IF NOT EXISTS idx_payment_transactions_provider_status ON payment_transactions(provider_name, status);

-- =============================================================================
-- NOTIFICATION AND EMAIL TABLES
-- =============================================================================

-- Email queue table
CREATE TABLE IF NOT EXISTS email_queue (
    id TEXT PRIMARY KEY,
    to_address TEXT NOT NULL,
    subject TEXT NOT NULL,
    body_html TEXT NOT NULL,
    body_text TEXT,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'sent', 'failed')),
    attempts INTEGER NOT NULL DEFAULT 0,
    last_error TEXT,
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    sent_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_email_queue_status ON email_queue(status);

-- Renewal reminders table
CREATE TABLE IF NOT EXISTS renewal_reminders (
    id TEXT PRIMARY KEY,
    license_id TEXT NOT NULL REFERENCES licenses(id) ON DELETE CASCADE,
    reminder_type TEXT NOT NULL CHECK (reminder_type IN ('30d', '14d', '7d', '1d')),
    sent_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_license_reminder_type UNIQUE (license_id, reminder_type)
);

CREATE INDEX IF NOT EXISTS idx_renewal_reminders_license_id ON renewal_reminders(license_id);
CREATE INDEX IF NOT EXISTS idx_renewal_reminders_type ON renewal_reminders(reminder_type);
CREATE INDEX IF NOT EXISTS idx_renewal_reminders_license_type ON renewal_reminders(license_id, reminder_type);

-- Notifications table
CREATE TABLE IF NOT EXISTS notifications (
    id TEXT PRIMARY KEY,
    user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    title TEXT NOT NULL,
    message TEXT NOT NULL,
    read BOOLEAN NOT NULL DEFAULT FALSE,
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications(user_id);

-- =============================================================================
-- ADDON AND FEATURE TABLES
-- =============================================================================

-- License addons table
CREATE TABLE IF NOT EXISTS license_addons (
    id TEXT PRIMARY KEY,
    license_id TEXT NOT NULL REFERENCES licenses(id) ON DELETE CASCADE,
    addon_key TEXT NOT NULL,
    quantity INTEGER NOT NULL DEFAULT 1,
    valid_from TIMESTAMP,
    valid_until TIMESTAMP,
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_license_addons_license_id ON license_addons(license_id);

-- =============================================================================
-- RELEASES AND ARTIFACTS
-- =============================================================================

-- Releases table
CREATE TABLE IF NOT EXISTS releases (
    id TEXT PRIMARY KEY,
    product_id TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    version TEXT NOT NULL,
    channel TEXT NOT NULL DEFAULT 'stable' CHECK (channel IN ('stable', 'beta', 'alpha')),
    release_notes TEXT,
    published_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_product_version UNIQUE (product_id, version)
);

CREATE INDEX IF NOT EXISTS idx_releases_product_id ON releases(product_id);

-- Release artifacts table
CREATE TABLE IF NOT EXISTS release_artifacts (
    id TEXT PRIMARY KEY,
    release_id TEXT NOT NULL REFERENCES releases(id) ON DELETE CASCADE,
    filename TEXT NOT NULL,
    platform TEXT NOT NULL,
    arch TEXT NOT NULL,
    download_url TEXT NOT NULL,
    file_size BIGINT,
    checksum TEXT,
    signature TEXT,
    tauri_signature TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_release_platform_arch UNIQUE (release_id, platform, arch)
);

CREATE INDEX IF NOT EXISTS idx_release_artifacts_release_id ON release_artifacts(release_id);

-- Release signing keys table
CREATE TABLE IF NOT EXISTS release_signing_keys (
    id TEXT PRIMARY KEY,
    product_id TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    public_key TEXT NOT NULL,
    key_type TEXT NOT NULL CHECK (key_type IN ('ed25519', 'rsa')),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deactivated_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_release_signing_keys_product_id ON release_signing_keys(product_id);

-- =============================================================================
-- AUTHENTICATION AND SECURITY
-- =============================================================================

-- Refresh tokens table
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);

-- OTP codes table
CREATE TABLE IF NOT EXISTS otp_codes (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL,
    code_hash TEXT NOT NULL,
    purpose TEXT NOT NULL CHECK (purpose IN ('login', 'verify', 'reset')),
    expires_at TIMESTAMP NOT NULL,
    used_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_otp_codes_email ON otp_codes(email);

-- =============================================================================
-- SYSTEM TABLES
-- =============================================================================

-- Settings table
CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Processed events table (idempotency)
CREATE TABLE IF NOT EXISTS processed_events (
    event_id TEXT PRIMARY KEY,
    event_type TEXT NOT NULL,
    processed_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_processed_events_type ON processed_events(event_type);

-- =============================================================================
-- COMMENTS
-- =============================================================================

COMMENT ON TABLE users IS 'User accounts for dashboard access';
COMMENT ON TABLE products IS 'Software products managed by Keygate';
COMMENT ON TABLE plans IS 'Pricing plans for products with multi-currency support';
COMMENT ON TABLE licenses IS 'License keys issued to customers';
COMMENT ON TABLE activations IS 'Device activations for licenses';
COMMENT ON TABLE payment_transactions IS 'Payment transaction records from various providers';
COMMENT ON TABLE email_queue IS 'Outbound email queue';
COMMENT ON TABLE renewal_reminders IS 'Tracks which renewal reminder emails have been sent to prevent duplicates';
COMMENT ON TABLE notifications IS 'In-app notifications for users';
COMMENT ON TABLE license_addons IS 'Add-ons attached to licenses';
COMMENT ON TABLE releases IS 'Software releases and versions';
COMMENT ON TABLE release_artifacts IS 'Downloadable artifacts for releases';
COMMENT ON TABLE release_signing_keys IS 'Public keys for verifying release signatures';
COMMENT ON TABLE refresh_tokens IS 'JWT refresh tokens for authentication';
COMMENT ON TABLE otp_codes IS 'One-time password codes for authentication';
COMMENT ON TABLE settings IS 'Global system settings';
COMMENT ON TABLE processed_events IS 'Webhook event deduplication tracking';
