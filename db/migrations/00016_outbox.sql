-- +goose Up
-- Outbox de eventos (§5.4). El worker goroutine pollorea eventos pendientes.

CREATE TABLE IF NOT EXISTS outbox_events (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    correlation_id UUID NOT NULL,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}',
    estado TEXT NOT NULL DEFAULT 'pendiente',
    intentos INT NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, id),
    CONSTRAINT outbox_events_estado_check CHECK (estado IN ('pendiente','procesado','fallido')),
    CONSTRAINT outbox_events_event_type_check CHECK (length(event_type) BETWEEN 1 AND 100),
    CONSTRAINT outbox_events_intentos_check CHECK (intentos >= 0)
);

CREATE INDEX IF NOT EXISTS outbox_events_pendientes_idx
    ON outbox_events(tenant_id, estado, next_attempt_at)
    WHERE estado = 'pendiente';

ALTER TABLE outbox_events ENABLE ROW LEVEL SECURITY;
CREATE POLICY outbox_events_visible ON outbox_events FOR SELECT USING (tenant_id = app.current_tenant_id());
CREATE POLICY outbox_events_insert ON outbox_events FOR INSERT WITH CHECK (tenant_id = app.current_tenant_id());
CREATE POLICY outbox_events_update ON outbox_events FOR UPDATE USING (tenant_id = app.current_tenant_id());

GRANT SELECT, INSERT, UPDATE ON outbox_events TO consorcio_app;

-- +goose Down
REVOKE SELECT, INSERT, UPDATE ON outbox_events FROM consorcio_app;
DROP POLICY IF EXISTS outbox_events_update ON outbox_events;
DROP POLICY IF EXISTS outbox_events_insert ON outbox_events;
DROP POLICY IF EXISTS outbox_events_visible ON outbox_events;
DROP TABLE IF EXISTS outbox_events;
