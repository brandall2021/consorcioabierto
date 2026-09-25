-- +goose Up
-- Cobros manuales/importación (H4.1, seccion 5.3).

CREATE TABLE IF NOT EXISTS payments (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    unidad_id UUID NOT NULL,
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    fecha DATE NOT NULL,
    canal TEXT NOT NULL DEFAULT 'otros',
    importe_cents BIGINT NOT NULL,
    referencia TEXT,
    estado TEXT NOT NULL DEFAULT 'pendiente_revision',
    idem_key TEXT NOT NULL,
    motivo_rechazo TEXT,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, id),
    FOREIGN KEY (tenant_id, unidad_id) REFERENCES unidades(tenant_id, id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT payments_canal_check CHECK (canal IN ('efectivo', 'transferencia', 'deposito', 'tarjeta', 'cajero', 'otros')),
    CONSTRAINT payments_estado_check CHECK (estado IN ('pendiente_revision', 'acreditado', 'rechazado', 'revertido')),
    CONSTRAINT payments_importe_check CHECK (importe_cents > 0),
    CONSTRAINT payments_idem_key_key UNIQUE (tenant_id, idem_key)
);

CREATE INDEX IF NOT EXISTS payments_unidad_idx ON payments(tenant_id, unidad_id, created_at DESC, id DESC);
CREATE UNIQUE INDEX IF NOT EXISTS payments_referencia_unique_idx
    ON payments(tenant_id, referencia)
    WHERE referencia IS NOT NULL;

ALTER TABLE payments ENABLE ROW LEVEL SECURITY;
CREATE POLICY payments_visible ON payments FOR SELECT USING (tenant_id = app.current_tenant_id());
CREATE POLICY payments_insert ON payments FOR INSERT WITH CHECK (tenant_id = app.current_tenant_id());
CREATE POLICY payments_update ON payments FOR UPDATE USING (tenant_id = app.current_tenant_id());

GRANT SELECT, INSERT, UPDATE ON payments TO consorcio_app;

-- +goose Down
REVOKE SELECT, INSERT, UPDATE ON payments FROM consorcio_app;
DROP POLICY IF EXISTS payments_update ON payments;
DROP POLICY IF EXISTS payments_insert ON payments;
DROP POLICY IF EXISTS payments_visible ON payments;
DROP TABLE IF EXISTS payments;
