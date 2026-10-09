-- Rollback multi-currency pricing columns from plans table
ALTER TABLE plans
    DROP COLUMN IF EXISTS stripe_price_id_hkd,
    DROP COLUMN IF EXISTS stripe_price_id_cny,
    DROP COLUMN IF EXISTS price_hkd,
    DROP COLUMN IF EXISTS price_cny;
