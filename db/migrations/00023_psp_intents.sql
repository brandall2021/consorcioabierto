-- +goose Up
CREATE TABLE IF NOT EXISTS psp_intents (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    payment_id UUID NOT NULL,
    provider TEXT NOT NULL,
    preference_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'created',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, id),
    FOREIGN KEY (tenant_id, payment_id) REFERENCES payments(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT psp_intents_provider_check CHECK (provider IN ('mercadopago', 'mock')),
    CONSTRAINT psp_intents_status_check CHECK (status IN ('created', 'approved', 'rejected', 'pending', 'cancelled')),
    CONSTRAINT psp_intents_preference_unique UNIQUE (tenant_id, preference_id),
    CONSTRAINT psp_intents_payment_unique UNIQUE (tenant_id, payment_id)
);

ALTER TABLE psp_intents ENABLE ROW LEVEL SECURITY;
CREATE POLICY psp_intents_visible ON psp_intents FOR SELECT USING (tenant_id = app.current_tenant_id());
CREATE POLICY psp_intents_insert ON psp_intents FOR INSERT WITH CHECK (tenant_id = app.current_tenant_id());
CREATE POLICY psp_intents_update ON psp_intents FOR UPDATE USING (tenant_id = app.current_tenant_id());

GRANT SELECT, INSERT, UPDATE ON psp_intents TO consorcio_app;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.find_psp_intent_by_preference_id(pref TEXT)
RETURNS TABLE (
    tenant_id UUID,
    payment_id UUID,
    created_by UUID,
    payment_estado TEXT,
    provider TEXT,
    preference_id TEXT,
    status TEXT
)
LANGUAGE sql
SECURITY DEFINER
SET search_path = public
AS $$
    SELECT i.tenant_id, i.payment_id, p.created_by, p.estado, i.provider, i.preference_id, i.status
    FROM psp_intents i
    JOIN payments p ON p.tenant_id = i.tenant_id AND p.id = i.payment_id
    WHERE i.preference_id = pref
    LIMIT 1;
$$;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION IF EXISTS app.find_psp_intent_by_preference_id(TEXT);
REVOKE SELECT, INSERT, UPDATE ON psp_intents FROM consorcio_app;
DROP POLICY IF EXISTS psp_intents_update ON psp_intents;
DROP POLICY IF EXISTS psp_intents_insert ON psp_intents;
DROP POLICY IF EXISTS psp_intents_visible ON psp_intents;
DROP TABLE IF EXISTS psp_intents;
