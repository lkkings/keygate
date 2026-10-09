-- payment_transactions table for multi-provider payment tracking
CREATE TABLE payment_transactions (
    id TEXT PRIMARY KEY,
    provider TEXT NOT NULL CHECK (provider IN ('stripe', 'alipay', 'wechat', 'epay')),
    provider_transaction_id TEXT NOT NULL,
    license_id TEXT REFERENCES licenses(id) ON DELETE SET NULL,
    plan_id TEXT REFERENCES plans(id) ON DELETE SET NULL,
    amount INTEGER NOT NULL,
    currency TEXT NOT NULL DEFAULT 'USD',
    status TEXT NOT NULL CHECK (status IN ('pending', 'completed', 'failed', 'refunded')),
    payment_method TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_provider_transaction UNIQUE (provider, provider_transaction_id)
);

-- Index for finding transactions by license
CREATE INDEX idx_payment_transactions_license_id ON payment_transactions(license_id);

-- Index for filtering by provider and status
CREATE INDEX idx_payment_transactions_provider_status ON payment_transactions(provider, status);

-- Index for time-based queries
CREATE INDEX idx_payment_transactions_created_at ON payment_transactions(created_at DESC);
