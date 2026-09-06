-- +goose Up
-- Conceptos de expensa (H3.1, §5.1). consorcio_id NULL = concepto de tenant
-- (usable por todos sus consorcios); no NULL = propio del consorcio.
CREATE TABLE IF NOT EXISTS conceptos_expensa (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    consorcio_id UUID,
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    nombre TEXT NOT NULL,
    categoria TEXT NOT NULL,
    regla TEXT NOT NULL DEFAULT 'coeficiente',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, id),
    CONSTRAINT conceptos_expensa_consorcio_fk
        FOREIGN KEY (tenant_id, consorcio_id) REFERENCES consorcios(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT conceptos_expensa_nombre_check CHECK (length(nombre) BETWEEN 1 AND 100),
    CONSTRAINT conceptos_expensa_categoria_check CHECK (length(categoria) BETWEEN 1 AND 50),
    CONSTRAINT conceptos_expensa_regla_check CHECK (regla IN ('coeficiente'))
);

CREATE UNIQUE INDEX IF NOT EXISTS conceptos_expensa_tenant_nombre_idx
    ON conceptos_expensa(tenant_id, consorcio_id, nombre) NULLS NOT DISTINCT;
CREATE INDEX IF NOT EXISTS conceptos_expensa_consorcio_idx
    ON conceptos_expensa(tenant_id, consorcio_id);

ALTER TABLE conceptos_expensa ENABLE ROW LEVEL SECURITY;
CREATE POLICY conceptos_expensa_visible ON conceptos_expensa FOR SELECT
    USING (tenant_id = app.current_tenant_id());
CREATE POLICY conceptos_expensa_insert ON conceptos_expensa FOR INSERT
    WITH CHECK (tenant_id = app.current_tenant_id());
CREATE POLICY conceptos_expensa_update ON conceptos_expensa FOR UPDATE
    USING (tenant_id = app.current_tenant_id());

GRANT SELECT, INSERT, UPDATE ON conceptos_expensa TO consorcio_app;

-- Gastos con comprobante y documento opcionales (H3.1, §5.1).
CREATE TABLE IF NOT EXISTS gastos (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    consorcio_id UUID NOT NULL,
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    concepto_id UUID NOT NULL,
    proveedor_id UUID,
    comprobante TEXT,
    importe_cents BIGINT NOT NULL,
    fecha DATE NOT NULL,
    estado TEXT NOT NULL DEFAULT 'registrado',
    documento_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, id),
    FOREIGN KEY (tenant_id, consorcio_id) REFERENCES consorcios(tenant_id, id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id, concepto_id) REFERENCES conceptos_expensa(tenant_id, id),
    FOREIGN KEY (tenant_id, proveedor_id) REFERENCES proveedores(tenant_id, id),
    CONSTRAINT gastos_importe_check CHECK (importe_cents > 0),
    CONSTRAINT gastos_estado_check CHECK (estado IN ('registrado', 'pagado', 'anulado')),
    CONSTRAINT gastos_comprobante_check CHECK (comprobante IS NULL OR length(comprobante) BETWEEN 1 AND 100),
    CONSTRAINT gastos_documento_fk
        FOREIGN KEY (tenant_id, documento_id) REFERENCES documentos(tenant_id, id)
);

CREATE INDEX IF NOT EXISTS gastos_consorcio_idx ON gastos(tenant_id, consorcio_id);
CREATE INDEX IF NOT EXISTS gastos_concepto_idx ON gastos(tenant_id, concepto_id);
CREATE INDEX IF NOT EXISTS gastos_fecha_idx ON gastos(tenant_id, consorcio_id, fecha DESC);

ALTER TABLE gastos ENABLE ROW LEVEL SECURITY;
CREATE POLICY gastos_visible ON gastos FOR SELECT USING (tenant_id = app.current_tenant_id());
CREATE POLICY gastos_insert ON gastos FOR INSERT WITH CHECK (tenant_id = app.current_tenant_id());
CREATE POLICY gastos_update ON gastos FOR UPDATE USING (tenant_id = app.current_tenant_id());

GRANT SELECT, INSERT, UPDATE ON gastos TO consorcio_app;

-- +goose Down
REVOKE SELECT, INSERT, UPDATE ON gastos FROM consorcio_app;
DROP POLICY IF EXISTS gastos_update ON gastos;
DROP POLICY IF EXISTS gastos_insert ON gastos;
DROP POLICY IF EXISTS gastos_visible ON gastos;
DROP TABLE IF EXISTS gastos;

REVOKE SELECT, INSERT, UPDATE ON conceptos_expensa FROM consorcio_app;
DROP POLICY IF EXISTS conceptos_expensa_update ON conceptos_expensa;
DROP POLICY IF EXISTS conceptos_expensa_insert ON conceptos_expensa;
DROP POLICY IF EXISTS conceptos_expensa_visible ON conceptos_expensa;
DROP TABLE IF EXISTS conceptos_expensa;