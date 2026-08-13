-- +goose Up
-- Consorcios: entidad raíz de organización por tenant. La PK compuesta
-- (tenant_id, id) sigue el patrón del ERD y permite FKs compuestas desde
-- unidades. RLS aísla por tenant y la app inserta/actualiza datos de negocio.

CREATE TABLE IF NOT EXISTS consorcios (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    nombre TEXT NOT NULL,
    cuit TEXT,
    domicilio TEXT,
    tipo TEXT NOT NULL DEFAULT 'edificio',
    estado TEXT NOT NULL DEFAULT 'activo',
    config JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, id),
    CONSTRAINT consorcios_nombre_check CHECK (length(nombre) BETWEEN 1 AND 200),
    CONSTRAINT consorcios_cuit_check CHECK (cuit IS NULL OR cuit ~ '^[0-9]{11}$'),
    CONSTRAINT consorcios_tipo_check CHECK (tipo IN ('edificio','barrio','complejo','otros')),
    CONSTRAINT consorcios_estado_check CHECK (estado IN ('activo','inactivo'))
);

CREATE UNIQUE INDEX IF NOT EXISTS consorcios_tenant_nombre_idx ON consorcios(tenant_id, lower(nombre));

ALTER TABLE consorcios ENABLE ROW LEVEL SECURITY;
CREATE POLICY consorcios_visible ON consorcios FOR SELECT USING (tenant_id = app.current_tenant_id());
CREATE POLICY consorcios_insert ON consorcios FOR INSERT WITH CHECK (tenant_id = app.current_tenant_id());
CREATE POLICY consorcios_update ON consorcios FOR UPDATE USING (tenant_id = app.current_tenant_id());

GRANT SELECT, INSERT, UPDATE ON consorcios TO consorcio_app;

-- +goose Down
REVOKE SELECT, INSERT, UPDATE ON consorcios FROM consorcio_app;

DROP POLICY IF EXISTS consorcios_update ON consorcios;
DROP POLICY IF EXISTS consorcios_insert ON consorcios;
DROP POLICY IF EXISTS consorcios_visible ON consorcios;

DROP TABLE IF EXISTS consorcios;
