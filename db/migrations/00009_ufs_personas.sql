-- +goose Up
-- Unidades funcionales, personas y vínculos históricos (ERD §5.1). PK compuesta
-- (tenant_id, id) en cada tabla para impedir referencias cruzadas entre tenants.
-- Los vínculos nunca se sobreescriben: se cierra valid_to y se crea otro registro.

CREATE TABLE IF NOT EXISTS unidades (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    consorcio_id UUID NOT NULL,
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    codigo TEXT NOT NULL,
    tipo TEXT NOT NULL,
    superficie NUMERIC(12,4),
    coeficiente NUMERIC(12,8) NOT NULL DEFAULT 0,
    estado TEXT NOT NULL DEFAULT 'activa',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, id),
    FOREIGN KEY (tenant_id, consorcio_id) REFERENCES consorcios(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT unidades_codigo_check CHECK (length(codigo) BETWEEN 1 AND 50),
    CONSTRAINT unidades_tipo_check CHECK (tipo IN ('departamento','cochera','local','unidad_edificio','otros')),
    CONSTRAINT unidades_estado_check CHECK (estado IN ('activa','inactiva')),
    CONSTRAINT unidades_coeficiente_check CHECK (coeficiente >= 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS unidades_consorcio_codigo_idx ON unidades(tenant_id, consorcio_id, codigo);

CREATE TABLE IF NOT EXISTS personas (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    nombre TEXT NOT NULL,
    documento TEXT,
    email TEXT,
    telefono TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, id),
    CONSTRAINT personas_nombre_check CHECK (length(nombre) BETWEEN 1 AND 200)
);

CREATE UNIQUE INDEX IF NOT EXISTS personas_documento_idx ON personas(tenant_id, documento) WHERE documento IS NOT NULL;

CREATE TABLE IF NOT EXISTS unidad_personas (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    unidad_id UUID NOT NULL,
    persona_id UUID NOT NULL,
    vinculo TEXT NOT NULL,
    porcentaje NUMERIC(5,2),
    valid_from DATE NOT NULL DEFAULT CURRENT_DATE,
    valid_to DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, id),
    FOREIGN KEY (tenant_id, unidad_id) REFERENCES unidades(tenant_id, id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id, persona_id) REFERENCES personas(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT unidad_personas_vinculo_check CHECK (vinculo IN ('propietario','inquilino','apoderado')),
    CONSTRAINT unidad_personas_porcentaje_check CHECK (porcentaje IS NULL OR (porcentaje >= 0 AND porcentaje <= 100)),
    CONSTRAINT unidad_personas_vigencia_check CHECK (valid_to IS NULL OR valid_to > valid_from)
);

CREATE INDEX IF NOT EXISTS unidad_personas_unidad_idx ON unidad_personas(tenant_id, unidad_id);
CREATE INDEX IF NOT EXISTS unidad_personas_vigente_idx ON unidad_personas(tenant_id, unidad_id) WHERE valid_to IS NULL;

ALTER TABLE unidades ENABLE ROW LEVEL SECURITY;
CREATE POLICY unidades_visible ON unidades FOR SELECT USING (tenant_id = app.current_tenant_id());
CREATE POLICY unidades_insert ON unidades FOR INSERT WITH CHECK (tenant_id = app.current_tenant_id());
CREATE POLICY unidades_update ON unidades FOR UPDATE USING (tenant_id = app.current_tenant_id());

ALTER TABLE personas ENABLE ROW LEVEL SECURITY;
CREATE POLICY personas_visible ON personas FOR SELECT USING (tenant_id = app.current_tenant_id());
CREATE POLICY personas_insert ON personas FOR INSERT WITH CHECK (tenant_id = app.current_tenant_id());
CREATE POLICY personas_update ON personas FOR UPDATE USING (tenant_id = app.current_tenant_id());

ALTER TABLE unidad_personas ENABLE ROW LEVEL SECURITY;
CREATE POLICY unidad_personas_visible ON unidad_personas FOR SELECT USING (tenant_id = app.current_tenant_id());
CREATE POLICY unidad_personas_insert ON unidad_personas FOR INSERT WITH CHECK (tenant_id = app.current_tenant_id());
CREATE POLICY unidad_personas_update ON unidad_personas FOR UPDATE USING (tenant_id = app.current_tenant_id());

GRANT SELECT, INSERT, UPDATE ON unidades TO consorcio_app;
GRANT SELECT, INSERT, UPDATE ON personas TO consorcio_app;
GRANT SELECT, INSERT, UPDATE ON unidad_personas TO consorcio_app;

-- +goose Down
REVOKE SELECT, INSERT, UPDATE ON unidades FROM consorcio_app;
REVOKE SELECT, INSERT, UPDATE ON personas FROM consorcio_app;
REVOKE SELECT, INSERT, UPDATE ON unidad_personas FROM consorcio_app;

DROP POLICY IF EXISTS unidad_personas_update ON unidad_personas;
DROP POLICY IF EXISTS unidad_personas_insert ON unidad_personas;
DROP POLICY IF EXISTS unidad_personas_visible ON unidad_personas;
DROP TABLE IF EXISTS unidad_personas;

DROP POLICY IF EXISTS personas_update ON personas;
DROP POLICY IF EXISTS personas_insert ON personas;
DROP POLICY IF EXISTS personas_visible ON personas;
DROP TABLE IF EXISTS personas;

DROP POLICY IF EXISTS unidades_update ON unidades;
DROP POLICY IF EXISTS unidades_insert ON unidades;
DROP POLICY IF EXISTS unidades_visible ON unidades;
DROP TABLE IF EXISTS unidades;