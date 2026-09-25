-- +goose Up
-- Comunicados y reclamos (Fase 5: portal y operación).

CREATE TABLE IF NOT EXISTS comunicados (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    consorcio_id UUID NOT NULL,
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    titulo TEXT NOT NULL,
    cuerpo TEXT NOT NULL,
    destinatarios TEXT NOT NULL DEFAULT 'todos',
    estado TEXT NOT NULL DEFAULT 'borrador',
    publicado_at TIMESTAMPTZ,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, id),
    FOREIGN KEY (tenant_id, consorcio_id) REFERENCES consorcios(tenant_id, id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT comunicados_destinatarios_check CHECK (destinatarios IN ('todos', 'unidades')),
    CONSTRAINT comunicados_estado_check CHECK (estado IN ('borrador', 'publicado'))
);

CREATE INDEX IF NOT EXISTS comunicados_consorcio_idx ON comunicados(tenant_id, consorcio_id, created_at DESC, id DESC);

ALTER TABLE comunicados ENABLE ROW LEVEL SECURITY;
CREATE POLICY comunicados_visible ON comunicados FOR SELECT USING (tenant_id = app.current_tenant_id());
CREATE POLICY comunicados_insert ON comunicados FOR INSERT WITH CHECK (tenant_id = app.current_tenant_id());
CREATE POLICY comunicados_update ON comunicados FOR UPDATE USING (tenant_id = app.current_tenant_id());

GRANT SELECT, INSERT, UPDATE ON comunicados TO consorcio_app;

CREATE TABLE IF NOT EXISTS reclamos (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    consorcio_id UUID NOT NULL,
    unidad_id UUID NOT NULL,
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    categoria TEXT NOT NULL,
    texto TEXT NOT NULL,
    estado TEXT NOT NULL DEFAULT 'abierto',
    responsable_id UUID,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, id),
    FOREIGN KEY (tenant_id, consorcio_id) REFERENCES consorcios(tenant_id, id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id, unidad_id) REFERENCES unidades(tenant_id, id) ON DELETE CASCADE,
    FOREIGN KEY (responsable_id) REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT reclamos_estado_check CHECK (estado IN ('abierto', 'en_progreso', 'resuelto', 'cerrado'))
);

CREATE INDEX IF NOT EXISTS reclamos_consorcio_idx ON reclamos(tenant_id, consorcio_id, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS reclamos_unidad_idx ON reclamos(tenant_id, unidad_id, created_at DESC, id DESC);

ALTER TABLE reclamos ENABLE ROW LEVEL SECURITY;
CREATE POLICY reclamos_visible ON reclamos FOR SELECT USING (tenant_id = app.current_tenant_id());
CREATE POLICY reclamos_insert ON reclamos FOR INSERT WITH CHECK (tenant_id = app.current_tenant_id());
CREATE POLICY reclamos_update ON reclamos FOR UPDATE USING (tenant_id = app.current_tenant_id());

GRANT SELECT, INSERT, UPDATE ON reclamos TO consorcio_app;

CREATE TABLE IF NOT EXISTS reclamo_mensajes (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    reclamo_id UUID NOT NULL,
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    texto TEXT NOT NULL,
    adjunto_id UUID,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, id),
    FOREIGN KEY (tenant_id, reclamo_id) REFERENCES reclamos(tenant_id, id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id, adjunto_id) REFERENCES documentos(tenant_id, id) ON DELETE SET NULL,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS reclamo_mensajes_reclamo_idx ON reclamo_mensajes(tenant_id, reclamo_id, created_at ASC, id ASC);

ALTER TABLE reclamo_mensajes ENABLE ROW LEVEL SECURITY;
CREATE POLICY reclamo_mensajes_visible ON reclamo_mensajes FOR SELECT USING (tenant_id = app.current_tenant_id());
CREATE POLICY reclamo_mensajes_insert ON reclamo_mensajes FOR INSERT WITH CHECK (tenant_id = app.current_tenant_id());

GRANT SELECT, INSERT ON reclamo_mensajes TO consorcio_app;

CREATE TABLE IF NOT EXISTS reclamo_transiciones (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    reclamo_id UUID NOT NULL,
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    accion TEXT NOT NULL,
    motivo TEXT,
    responsable_id UUID,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, id),
    FOREIGN KEY (tenant_id, reclamo_id) REFERENCES reclamos(tenant_id, id) ON DELETE CASCADE,
    FOREIGN KEY (responsable_id) REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT reclamo_transiciones_accion_check CHECK (accion IN ('asignar', 'en_progreso', 'resolver', 'cerrar', 'reabrir'))
);

CREATE INDEX IF NOT EXISTS reclamo_transiciones_reclamo_idx ON reclamo_transiciones(tenant_id, reclamo_id, created_at ASC, id ASC);

ALTER TABLE reclamo_transiciones ENABLE ROW LEVEL SECURITY;
CREATE POLICY reclamo_transiciones_visible ON reclamo_transiciones FOR SELECT USING (tenant_id = app.current_tenant_id());
CREATE POLICY reclamo_transiciones_insert ON reclamo_transiciones FOR INSERT WITH CHECK (tenant_id = app.current_tenant_id());

GRANT SELECT, INSERT ON reclamo_transiciones TO consorcio_app;

-- +goose Down
REVOKE SELECT, INSERT ON reclamo_transiciones FROM consorcio_app;
REVOKE SELECT, INSERT ON reclamo_mensajes FROM consorcio_app;
REVOKE SELECT, INSERT, UPDATE ON reclamos FROM consorcio_app;
REVOKE SELECT, INSERT, UPDATE ON comunicados FROM consorcio_app;
DROP POLICY IF EXISTS reclamo_transiciones_insert ON reclamo_transiciones;
DROP POLICY IF EXISTS reclamo_transiciones_visible ON reclamo_transiciones;
DROP TABLE IF EXISTS reclamo_transiciones;
DROP POLICY IF EXISTS reclamo_mensajes_insert ON reclamo_mensajes;
DROP POLICY IF EXISTS reclamo_mensajes_visible ON reclamo_mensajes;
DROP TABLE IF EXISTS reclamo_mensajes;
DROP POLICY IF EXISTS reclamos_update ON reclamos;
DROP POLICY IF EXISTS reclamos_insert ON reclamos;
DROP POLICY IF EXISTS reclamos_visible ON reclamos;
DROP TABLE IF EXISTS reclamos;
DROP POLICY IF EXISTS comunicados_update ON comunicados;
DROP POLICY IF EXISTS comunicados_insert ON comunicados;
DROP POLICY IF EXISTS comunicados_visible ON comunicados;
DROP TABLE IF EXISTS comunicados;
