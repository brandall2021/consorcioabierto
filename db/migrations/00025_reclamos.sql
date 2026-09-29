-- +goose Up
CREATE TABLE IF NOT EXISTS reclamos (
    tenant_id      UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    id             UUID NOT NULL DEFAULT gen_random_uuid(),
    consorcio_id   UUID NOT NULL,
    unidad_id      UUID NOT NULL,
    categoria      TEXT NOT NULL,
    texto          TEXT NOT NULL,
    estado         TEXT NOT NULL DEFAULT 'abierto',
    responsable_id UUID REFERENCES users(id) ON DELETE SET NULL,
    sla_due_at     TIMESTAMPTZ,
    created_by     UUID DEFAULT app.current_user_id(),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, id),
    FOREIGN KEY (tenant_id, consorcio_id) REFERENCES consorcios(tenant_id, id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id, unidad_id) REFERENCES unidades(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT reclamos_estado_check CHECK (estado IN ('abierto', 'en_progreso', 'resuelto', 'cerrado')),
    CONSTRAINT reclamos_texto_check CHECK (length(texto) >= 1)
);

ALTER TABLE reclamos ENABLE ROW LEVEL SECURITY;
CREATE POLICY reclamos_visible ON reclamos FOR SELECT USING (tenant_id = app.current_tenant_id());
CREATE POLICY reclamos_insert ON reclamos FOR INSERT WITH CHECK (tenant_id = app.current_tenant_id());
CREATE POLICY reclamos_update ON reclamos FOR UPDATE USING (tenant_id = app.current_tenant_id());

GRANT SELECT, INSERT, UPDATE ON reclamos TO consorcio_app;

CREATE TABLE IF NOT EXISTS reclamo_mensajes (
    tenant_id  UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    id         UUID NOT NULL DEFAULT gen_random_uuid(),
    reclamo_id UUID NOT NULL,
    texto      TEXT NOT NULL,
    adjunto_id UUID,
    created_by UUID DEFAULT app.current_user_id(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, id),
    FOREIGN KEY (tenant_id, reclamo_id) REFERENCES reclamos(tenant_id, id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id, adjunto_id) REFERENCES documentos(tenant_id, id) ON DELETE SET NULL,
    CONSTRAINT reclamo_mensajes_texto_check CHECK (length(texto) >= 1)
);

ALTER TABLE reclamo_mensajes ENABLE ROW LEVEL SECURITY;
CREATE POLICY reclamo_mensajes_visible ON reclamo_mensajes FOR SELECT USING (tenant_id = app.current_tenant_id());
CREATE POLICY reclamo_mensajes_insert ON reclamo_mensajes FOR INSERT WITH CHECK (tenant_id = app.current_tenant_id());

GRANT SELECT, INSERT ON reclamo_mensajes TO consorcio_app;

CREATE TABLE IF NOT EXISTS reclamo_transiciones (
    tenant_id      UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    id             UUID NOT NULL DEFAULT gen_random_uuid(),
    reclamo_id     UUID NOT NULL,
    accion         TEXT NOT NULL,
    motivo         TEXT,
    from_estado    TEXT NOT NULL,
    to_estado      TEXT NOT NULL,
    responsable_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_by     UUID DEFAULT app.current_user_id(),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, id),
    FOREIGN KEY (tenant_id, reclamo_id) REFERENCES reclamos(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT reclamo_transiciones_accion_check CHECK (accion IN ('asignar', 'en_progreso', 'resolver', 'cerrar', 'reabrir')),
    CONSTRAINT reclamo_transiciones_estado_check CHECK (from_estado IN ('abierto', 'en_progreso', 'resuelto', 'cerrado') AND to_estado IN ('abierto', 'en_progreso', 'resuelto', 'cerrado'))
);

ALTER TABLE reclamo_transiciones ENABLE ROW LEVEL SECURITY;
CREATE POLICY reclamo_transiciones_visible ON reclamo_transiciones FOR SELECT USING (tenant_id = app.current_tenant_id());
CREATE POLICY reclamo_transiciones_insert ON reclamo_transiciones FOR INSERT WITH CHECK (tenant_id = app.current_tenant_id());

GRANT SELECT, INSERT ON reclamo_transiciones TO consorcio_app;

-- +goose Down
REVOKE SELECT, INSERT ON reclamo_transiciones FROM consorcio_app;
DROP POLICY IF EXISTS reclamo_transiciones_insert ON reclamo_transiciones;
DROP POLICY IF EXISTS reclamo_transiciones_visible ON reclamo_transiciones;
DROP TABLE IF EXISTS reclamo_transiciones;

REVOKE SELECT, INSERT ON reclamo_mensajes FROM consorcio_app;
DROP POLICY IF EXISTS reclamo_mensajes_insert ON reclamo_mensajes;
DROP POLICY IF EXISTS reclamo_mensajes_visible ON reclamo_mensajes;
DROP TABLE IF EXISTS reclamo_mensajes;

REVOKE SELECT, INSERT, UPDATE ON reclamos FROM consorcio_app;
DROP POLICY IF EXISTS reclamos_update ON reclamos;
DROP POLICY IF EXISTS reclamos_insert ON reclamos;
DROP POLICY IF EXISTS reclamos_visible ON reclamos;
DROP TABLE IF EXISTS reclamos;