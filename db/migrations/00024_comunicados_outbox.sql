-- +goose Up
CREATE TABLE IF NOT EXISTS comunicados (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    consorcio_id UUID NOT NULL,
    titulo TEXT NOT NULL,
    cuerpo TEXT NOT NULL,
    destinatarios_scope TEXT NOT NULL DEFAULT 'todos',
    destinatarios BIGINT NOT NULL DEFAULT 0,
    estado TEXT NOT NULL DEFAULT 'borrador',
    publicado_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, id),
    FOREIGN KEY (tenant_id, consorcio_id) REFERENCES consorcios(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT comunicados_estado_check CHECK (estado IN ('borrador', 'publicado')),
    CONSTRAINT comunicados_destinatarios_scope_check CHECK (destinatarios_scope IN ('todos', 'unidades')),
    CONSTRAINT comunicados_destinatarios_check CHECK (destinatarios >= 0)
);

ALTER TABLE comunicados ENABLE ROW LEVEL SECURITY;
CREATE POLICY comunicados_visible ON comunicados FOR SELECT USING (tenant_id = app.current_tenant_id());
CREATE POLICY comunicados_insert ON comunicados FOR INSERT WITH CHECK (tenant_id = app.current_tenant_id());
CREATE POLICY comunicados_update ON comunicados FOR UPDATE USING (tenant_id = app.current_tenant_id());

GRANT SELECT, INSERT, UPDATE ON comunicados TO consorcio_app;

CREATE TABLE IF NOT EXISTS outbox_events (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    correlation_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    estado TEXT NOT NULL DEFAULT 'pendiente',
    intentos INTEGER NOT NULL DEFAULT 0,
    last_error TEXT NOT NULL DEFAULT '',
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, id),
    CONSTRAINT outbox_events_estado_check CHECK (estado IN ('pendiente', 'procesando', 'procesado', 'fallido')),
    CONSTRAINT outbox_events_intentos_check CHECK (intentos >= 0)
);

ALTER TABLE outbox_events ENABLE ROW LEVEL SECURITY;
CREATE POLICY outbox_events_visible ON outbox_events FOR SELECT USING (tenant_id = app.current_tenant_id());
CREATE POLICY outbox_events_insert ON outbox_events FOR INSERT WITH CHECK (tenant_id = app.current_tenant_id());
CREATE POLICY outbox_events_update ON outbox_events FOR UPDATE USING (tenant_id = app.current_tenant_id());

GRANT SELECT, INSERT, UPDATE ON outbox_events TO consorcio_app;

CREATE OR REPLACE FUNCTION app.claim_pending_outbox_events(p_limit INTEGER)
RETURNS TABLE (
    tenant_id UUID,
    id UUID,
    correlation_id TEXT,
    event_type TEXT,
    payload JSONB,
    estado TEXT,
    intentos INTEGER,
    last_error TEXT,
    next_attempt_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
)
LANGUAGE sql
SECURITY DEFINER
SET search_path = public
AS $$
    WITH picked AS (
        SELECT tenant_id, id
        FROM outbox_events
        WHERE estado IN ('pendiente', 'fallido')
          AND intentos < 3
          AND next_attempt_at <= now()
        ORDER BY created_at ASC, id ASC
        LIMIT p_limit
        FOR UPDATE SKIP LOCKED
    )
    UPDATE outbox_events o
    SET estado = 'procesando',
        intentos = intentos + 1,
        updated_at = now()
    FROM picked
    WHERE o.tenant_id = picked.tenant_id AND o.id = picked.id
    RETURNING o.tenant_id, o.id, o.correlation_id, o.event_type, o.payload, o.estado,
              o.intentos, o.last_error, o.next_attempt_at, o.created_at, o.updated_at;
$$;

GRANT EXECUTE ON FUNCTION app.claim_pending_outbox_events(INTEGER) TO consorcio_app;

-- +goose Down
REVOKE EXECUTE ON FUNCTION app.claim_pending_outbox_events(INTEGER) FROM consorcio_app;
DROP FUNCTION IF EXISTS app.claim_pending_outbox_events(INTEGER);

REVOKE SELECT, INSERT, UPDATE ON outbox_events FROM consorcio_app;
DROP POLICY IF EXISTS outbox_events_update ON outbox_events;
DROP POLICY IF EXISTS outbox_events_insert ON outbox_events;
DROP POLICY IF EXISTS outbox_events_visible ON outbox_events;
DROP TABLE IF EXISTS outbox_events;

REVOKE SELECT, INSERT, UPDATE ON comunicados FROM consorcio_app;
DROP POLICY IF EXISTS comunicados_update ON comunicados;
DROP POLICY IF EXISTS comunicados_insert ON comunicados;
DROP POLICY IF EXISTS comunicados_visible ON comunicados;
DROP TABLE IF EXISTS comunicados;
