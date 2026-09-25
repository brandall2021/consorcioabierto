-- +goose Up
-- Asignaciones de cobros acreditados (H4.2, ADR-0007).

CREATE TABLE IF NOT EXISTS payment_allocations (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    payment_id UUID NOT NULL,
    charge_id UUID NOT NULL,
    amount_cents BIGINT NOT NULL,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, payment_id, charge_id),
    FOREIGN KEY (tenant_id, payment_id) REFERENCES payments(tenant_id, id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id, charge_id) REFERENCES charges(tenant_id, id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT payment_allocations_amount_check CHECK (amount_cents > 0)
);

CREATE INDEX IF NOT EXISTS payment_allocations_payment_idx ON payment_allocations(tenant_id, payment_id, created_at ASC, charge_id ASC);
CREATE INDEX IF NOT EXISTS payment_allocations_charge_idx ON payment_allocations(tenant_id, charge_id);

ALTER TABLE payment_allocations ENABLE ROW LEVEL SECURITY;
CREATE POLICY payment_allocations_visible ON payment_allocations FOR SELECT USING (tenant_id = app.current_tenant_id());
CREATE POLICY payment_allocations_insert ON payment_allocations FOR INSERT WITH CHECK (tenant_id = app.current_tenant_id());
CREATE POLICY payment_allocations_update ON payment_allocations FOR UPDATE USING (tenant_id = app.current_tenant_id());
CREATE POLICY payment_allocations_delete ON payment_allocations FOR DELETE USING (tenant_id = app.current_tenant_id());

GRANT SELECT, INSERT, UPDATE, DELETE ON payment_allocations TO consorcio_app;

-- +goose Down
REVOKE SELECT, INSERT, UPDATE, DELETE ON payment_allocations FROM consorcio_app;
DROP POLICY IF EXISTS payment_allocations_delete ON payment_allocations;
DROP POLICY IF EXISTS payment_allocations_update ON payment_allocations;
DROP POLICY IF EXISTS payment_allocations_insert ON payment_allocations;
DROP POLICY IF EXISTS payment_allocations_visible ON payment_allocations;
DROP TABLE IF EXISTS payment_allocations;
