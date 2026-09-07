-- +goose Up
-- Liquidaciones y snapshots (H3.2-H3.4, §5.2).

CREATE TABLE IF NOT EXISTS liquidaciones (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    consorcio_id UUID NOT NULL,
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    periodo TEXT NOT NULL, -- 'YYYYMM'
    vencimiento_1 DATE NOT NULL,
    vencimiento_2 DATE,
    estado TEXT NOT NULL DEFAULT 'borrador',
    version INT NOT NULL DEFAULT 1,
    total_gastos_cents BIGINT NOT NULL DEFAULT 0,
    total_distribuido_cents BIGINT NOT NULL DEFAULT 0,
    unidades_alcanzadas INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, id),
    CONSTRAINT liquidaciones_consorcio_fk
        FOREIGN KEY (tenant_id, consorcio_id) REFERENCES consorcios(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT liquidaciones_periodo_check CHECK (periodo ~ '^[0-9]{6}$'),
    CONSTRAINT liquidaciones_estado_check
        CHECK (estado IN ('borrador','calculada','confirmada','publicada','cerrada','anulada')),
    CONSTRAINT liquidaciones_version_check CHECK (version >= 1),
    CONSTRAINT liquidaciones_vencimientos_check CHECK (vencimiento_2 IS NULL OR vencimiento_2 >= vencimiento_1),
    CONSTRAINT liquidaciones_anulada_motivo_check
        CHECK (estado != 'anulada') -- motivos se registran en audit_events
);

-- Una liquidación activa por período; anulada/cerrada liberan la combinación.
CREATE UNIQUE INDEX IF NOT EXISTS liquidaciones_unica_activa_idx
    ON liquidaciones(tenant_id, consorcio_id, periodo)
    WHERE estado NOT IN ('anulada', 'cerrada');

CREATE INDEX IF NOT EXISTS liquidaciones_consorcio_idx ON liquidaciones(tenant_id, consorcio_id);
CREATE INDEX IF NOT EXISTS liquidaciones_estado_idx ON liquidaciones(tenant_id, consorcio_id, estado);

ALTER TABLE liquidaciones ENABLE ROW LEVEL SECURITY;
CREATE POLICY liquidaciones_visible ON liquidaciones FOR SELECT USING (tenant_id = app.current_tenant_id());
CREATE POLICY liquidaciones_insert ON liquidaciones FOR INSERT WITH CHECK (tenant_id = app.current_tenant_id());
CREATE POLICY liquidaciones_update ON liquidaciones FOR UPDATE USING (tenant_id = app.current_tenant_id());

GRANT SELECT, INSERT, UPDATE ON liquidaciones TO consorcio_app;

-- Snapshot de gastos incluidos en la liquidación.
CREATE TABLE IF NOT EXISTS liquidacion_gastos (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    liquidacion_id UUID NOT NULL,
    gasto_id UUID NOT NULL,
    concepto_id UUID NOT NULL,
    importe_cents BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, liquidacion_id, gasto_id),
    FOREIGN KEY (tenant_id, liquidacion_id) REFERENCES liquidaciones(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT liquidacion_gastos_importe_check CHECK (importe_cents > 0)
);

CREATE INDEX IF NOT EXISTS liquidacion_gastos_liquidacion_idx ON liquidacion_gastos(tenant_id, liquidacion_id);

ALTER TABLE liquidacion_gastos ENABLE ROW LEVEL SECURITY;
CREATE POLICY liquidacion_gastos_visible ON liquidacion_gastos FOR SELECT USING (tenant_id = app.current_tenant_id());
CREATE POLICY liquidacion_gastos_insert ON liquidacion_gastos FOR INSERT WITH CHECK (tenant_id = app.current_tenant_id());

GRANT SELECT, INSERT ON liquidacion_gastos TO consorcio_app;

-- Importe total por concepto y regla aplicada.
CREATE TABLE IF NOT EXISTS liquidacion_items (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    liquidacion_id UUID NOT NULL,
    concepto_id UUID NOT NULL,
    regla_aplicada TEXT NOT NULL,
    importe_cents BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, liquidacion_id, concepto_id),
    FOREIGN KEY (tenant_id, liquidacion_id) REFERENCES liquidaciones(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT liquidacion_items_importe_check CHECK (importe_cents >= 0)
);

CREATE INDEX IF NOT EXISTS liquidacion_items_liquidacion_idx ON liquidacion_items(tenant_id, liquidacion_id);

ALTER TABLE liquidacion_items ENABLE ROW LEVEL SECURITY;
CREATE POLICY liquidacion_items_visible ON liquidacion_items FOR SELECT USING (tenant_id = app.current_tenant_id());
CREATE POLICY liquidacion_items_insert ON liquidacion_items FOR INSERT WITH CHECK (tenant_id = app.current_tenant_id());

GRANT SELECT, INSERT ON liquidacion_items TO consorcio_app;

-- Snapshot de UF, coeficiente y total asignado.
CREATE TABLE IF NOT EXISTS liquidacion_unidades (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    liquidacion_id UUID NOT NULL,
    unidad_id UUID NOT NULL,
    codigo TEXT NOT NULL,
    coeficiente NUMERIC(12,8) NOT NULL,
    total_cents BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, liquidacion_id, unidad_id),
    FOREIGN KEY (tenant_id, liquidacion_id) REFERENCES liquidaciones(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT liquidacion_unidades_coeficiente_check CHECK (coeficiente >= 0),
    CONSTRAINT liquidacion_unidades_total_check CHECK (total_cents >= 0)
);

CREATE INDEX IF NOT EXISTS liquidacion_unidades_liquidacion_idx ON liquidacion_unidades(tenant_id, liquidacion_id);

ALTER TABLE liquidacion_unidades ENABLE ROW LEVEL SECURITY;
CREATE POLICY liquidacion_unidades_visible ON liquidacion_unidades FOR SELECT USING (tenant_id = app.current_tenant_id());
CREATE POLICY liquidacion_unidades_insert ON liquidacion_unidades FOR INSERT WITH CHECK (tenant_id = app.current_tenant_id());

GRANT SELECT, INSERT ON liquidacion_unidades TO consorcio_app;

-- Desglose por UF/concepto.
CREATE TABLE IF NOT EXISTS liquidacion_unidad_items (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    liquidacion_id UUID NOT NULL,
    liquidacion_unidad_id UUID NOT NULL,
    concepto_id UUID NOT NULL,
    importe_cents BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, liquidacion_id, liquidacion_unidad_id, concepto_id),
    FOREIGN KEY (tenant_id, liquidacion_id, liquidacion_unidad_id)
        REFERENCES liquidacion_unidades(tenant_id, liquidacion_id, unidad_id) ON DELETE CASCADE,
    CONSTRAINT liquidacion_unidad_items_importe_check CHECK (importe_cents >= 0)
);

CREATE INDEX IF NOT EXISTS liquidacion_unidad_items_idx ON liquidacion_unidad_items(tenant_id, liquidacion_id);

ALTER TABLE liquidacion_unidad_items ENABLE ROW LEVEL SECURITY;
CREATE POLICY liquidacion_unidad_items_visible ON liquidacion_unidad_items FOR SELECT USING (tenant_id = app.current_tenant_id());
CREATE POLICY liquidacion_unidad_items_insert ON liquidacion_unidad_items FOR INSERT WITH CHECK (tenant_id = app.current_tenant_id());

GRANT SELECT, INSERT ON liquidacion_unidad_items TO consorcio_app;

-- +goose Down
REVOKE SELECT, INSERT ON liquidacion_unidad_items FROM consorcio_app;
DROP POLICY IF EXISTS liquidacion_unidad_items_insert ON liquidacion_unidad_items;
DROP POLICY IF EXISTS liquidacion_unidad_items_visible ON liquidacion_unidad_items;
DROP TABLE IF EXISTS liquidacion_unidad_items;

REVOKE SELECT, INSERT ON liquidacion_unidades FROM consorcio_app;
DROP POLICY IF EXISTS liquidacion_unidades_insert ON liquidacion_unidades;
DROP POLICY IF EXISTS liquidacion_unidades_visible ON liquidacion_unidades;
DROP TABLE IF EXISTS liquidacion_unidades;

REVOKE SELECT, INSERT ON liquidacion_items FROM consorcio_app;
DROP POLICY IF EXISTS liquidacion_items_insert ON liquidacion_items;
DROP POLICY IF EXISTS liquidacion_items_visible ON liquidacion_items;
DROP TABLE IF EXISTS liquidacion_items;

REVOKE SELECT, INSERT ON liquidacion_gastos FROM consorcio_app;
DROP POLICY IF EXISTS liquidacion_gastos_insert ON liquidacion_gastos;
DROP POLICY IF EXISTS liquidacion_gastos_visible ON liquidacion_gastos;
DROP TABLE IF EXISTS liquidacion_gastos;

REVOKE SELECT, INSERT, UPDATE ON liquidaciones FROM consorcio_app;
DROP POLICY IF EXISTS liquidaciones_update ON liquidaciones;
DROP POLICY IF EXISTS liquidaciones_insert ON liquidaciones;
DROP POLICY IF EXISTS liquidaciones_visible ON liquidaciones;
DROP TABLE IF EXISTS liquidaciones;
