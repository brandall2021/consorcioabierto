-- +goose Up
-- Cuenta corriente por UF (H3.5, §5.3).

CREATE TABLE IF NOT EXISTS charges (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    unidad_id UUID NOT NULL,
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    liquidacion_id UUID,
    concepto TEXT NOT NULL,
    due_date DATE NOT NULL,
    total_cents BIGINT NOT NULL,
    saldo_cents BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, id),
    FOREIGN KEY (tenant_id, unidad_id) REFERENCES unidades(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT charges_total_check CHECK (total_cents > 0),
    CONSTRAINT charges_saldo_check CHECK (saldo_cents >= 0),
    CONSTRAINT charges_saldo_lte_total CHECK (saldo_cents <= total_cents)
);

CREATE INDEX IF NOT EXISTS charges_unidad_idx ON charges(tenant_id, unidad_id);
CREATE INDEX IF NOT EXISTS charges_liquidacion_idx ON charges(tenant_id, liquidacion_id);

ALTER TABLE charges ENABLE ROW LEVEL SECURITY;
CREATE POLICY charges_visible ON charges FOR SELECT USING (tenant_id = app.current_tenant_id());
CREATE POLICY charges_insert ON charges FOR INSERT WITH CHECK (tenant_id = app.current_tenant_id());
CREATE POLICY charges_update ON charges FOR UPDATE USING (tenant_id = app.current_tenant_id());

GRANT SELECT, INSERT, UPDATE ON charges TO consorcio_app;

CREATE TABLE IF NOT EXISTS account_entries (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    unidad_id UUID NOT NULL,
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    tipo TEXT NOT NULL,
    fecha_efectiva DATE NOT NULL,
    debit_cents BIGINT NOT NULL DEFAULT 0,
    credit_cents BIGINT NOT NULL DEFAULT 0,
    currency TEXT NOT NULL DEFAULT 'ARS',
    referencia TEXT,
    reversa_de_id UUID,
    charge_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, id),
    FOREIGN KEY (tenant_id, unidad_id) REFERENCES unidades(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT account_entries_tipo_check CHECK (tipo IN ('cargo','reversa','credito')),
    CONSTRAINT account_entries_xor_check
        CHECK ((debit_cents > 0 AND credit_cents = 0) OR (credit_cents > 0 AND debit_cents = 0)),
    CONSTRAINT account_entries_debit_check CHECK (debit_cents >= 0),
    CONSTRAINT account_entries_credit_check CHECK (credit_cents >= 0)
);

CREATE INDEX IF NOT EXISTS account_entries_unidad_idx ON account_entries(tenant_id, unidad_id);
CREATE INDEX IF NOT EXISTS account_entries_tipo_idx ON account_entries(tenant_id, unidad_id, tipo);

ALTER TABLE account_entries ENABLE ROW LEVEL SECURITY;
CREATE POLICY account_entries_visible ON account_entries FOR SELECT USING (tenant_id = app.current_tenant_id());
CREATE POLICY account_entries_insert ON account_entries FOR INSERT WITH CHECK (tenant_id = app.current_tenant_id());
CREATE POLICY account_entries_update ON account_entries FOR UPDATE USING (tenant_id = app.current_tenant_id());

GRANT SELECT, INSERT, UPDATE ON account_entries TO consorcio_app;

-- +goose Down
REVOKE SELECT, INSERT, UPDATE ON account_entries FROM consorcio_app;
DROP POLICY IF EXISTS account_entries_update ON account_entries;
DROP POLICY IF EXISTS account_entries_insert ON account_entries;
DROP POLICY IF EXISTS account_entries_visible ON account_entries;
DROP TABLE IF EXISTS account_entries;

REVOKE SELECT, INSERT, UPDATE ON charges FROM consorcio_app;
DROP POLICY IF EXISTS charges_update ON charges;
DROP POLICY IF EXISTS charges_insert ON charges;
DROP POLICY IF EXISTS charges_visible ON charges;
DROP TABLE IF EXISTS charges;
