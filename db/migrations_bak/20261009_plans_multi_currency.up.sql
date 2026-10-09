-- Add multi-currency pricing columns to plans table
ALTER TABLE plans
    ADD COLUMN IF NOT EXISTS price_cny INTEGER,
    ADD COLUMN IF NOT EXISTS price_hkd INTEGER,
    ADD COLUMN IF NOT EXISTS stripe_price_id_cny TEXT,
    ADD COLUMN IF NOT EXISTS stripe_price_id_hkd TEXT;

-- Add comment to clarify these are in cents/smallest unit
COMMENT ON COLUMN plans.price_cny IS 'Price in CNY cents (¥99.00 = 9900)';
COMMENT ON COLUMN plans.price_hkd IS 'Price in HKD cents (HK$99.00 = 9900)';
