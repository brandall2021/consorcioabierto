-- +goose Up
ALTER TABLE payments
    ADD COLUMN IF NOT EXISTS psp_provider TEXT,
    ADD COLUMN IF NOT EXISTS psp_preference_id TEXT,
    ADD COLUMN IF NOT EXISTS psp_checkout_url TEXT;

-- +goose Down
ALTER TABLE payments
    DROP COLUMN IF EXISTS psp_checkout_url,
    DROP COLUMN IF EXISTS psp_preference_id,
    DROP COLUMN IF EXISTS psp_provider;
