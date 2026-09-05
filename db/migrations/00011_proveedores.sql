-- +goose Up
-- Proveedores por consorcio (H2.4, §5.4). CUIT único dentro del mismo consorcio.
-- PK compuesta (tenant_id, id) como el resto de tablas tenant-scoped.

CREATE TABLE IF NOT EXISTS proveedores (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    consorcio_id UUID NOT NULL,
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    cuit TEXT NOT NULL,
    razon_social TEXT NOT NULL,
    contacto_nombre TEXT,
    contacto_email TEXT,
    contacto_telefono TEXT,
    estado TEXT NOT NULL DEFAULT 'activo',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, id),
    FOREIGN KEY (tenant_id, consorcio_id) REFERENCES consorcios(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT proveedores_cuit_check CHECK (cuit ~ '^[0-9]{11}$'),
    CONSTRAINT proveedores_razon_social_check CHECK (length(razon_social) BETWEEN 1 AND 200),
    CONSTRAINT proveedores_estado_check CHECK (estado IN ('activo', 'inactivo'))
);

CREATE UNIQUE INDEX IF NOT EXISTS proveedores_consorcio_cuit_idx ON proveedores(tenant_id, consorcio_id, cuit);

ALTER TABLE proveedores ENABLE ROW LEVEL SECURITY;
CREATE POLICY proveedores_visible ON proveedores FOR SELECT USING (tenant_id = app.current_tenant_id());
CREATE POLICY proveedores_insert ON proveedores FOR INSERT WITH CHECK (tenant_id = app.current_tenant_id());
CREATE POLICY proveedores_update ON proveedores FOR UPDATE USING (tenant_id = app.current_tenant_id());

GRANT SELECT, INSERT, UPDATE ON proveedores TO consorcio_app;

-- +goose Down
REVOKE SELECT, INSERT, UPDATE ON proveedores FROM consorcio_app;

DROP POLICY IF EXISTS proveedores_update ON proveedores;
DROP POLICY IF EXISTS proveedores_insert ON proveedores;
DROP POLICY IF EXISTS proveedores_visible ON proveedores;

DROP TABLE IF EXISTS proveedores;