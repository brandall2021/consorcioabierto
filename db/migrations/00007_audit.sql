-- +goose Up
-- Auditoría append-only.
--
-- audit_events registra acciones de negocio con contexto de auditoría
-- (actor, tenant, acción, recurso, request ID, IP, user-agent y diff seguro).
-- Es estrictamente append-only: consorcio_app solo puede INSERTAR a través de
-- la función SECURITY DEFINER app.record_audit_event (sin UPDATE/DELETE), y un
-- trigger bloquea UPDATE/DELETE/TRUNCATE incluso para el owner.

CREATE TABLE IF NOT EXISTS audit_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE SET NULL,
    actor_id UUID,
    actor_membership UUID,
    accion TEXT NOT NULL,
    recurso_type TEXT NOT NULL,
    recurso_id TEXT,
    request_id TEXT,
    ip TEXT,
    user_agent TEXT,
    diff JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS audit_events_tenant_created_idx ON audit_events(tenant_id, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS audit_events_actor_created_idx ON audit_events(actor_id, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS audit_events_accion_idx ON audit_events(accion);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.record_audit_event(
    p_tenant_id UUID,
    p_actor_id UUID,
    p_actor_membership UUID,
    p_accion TEXT,
    p_recurso_type TEXT,
    p_recurso_id TEXT,
    p_request_id TEXT,
    p_ip TEXT,
    p_user_agent TEXT,
    p_diff JSONB
) RETURNS uuid LANGUAGE plpgsql SECURITY DEFINER AS $$
DECLARE
    v_id UUID;
BEGIN
    INSERT INTO audit_events (tenant_id, actor_id, actor_membership, accion, recurso_type,
                              recurso_id, request_id, ip, user_agent, diff)
    VALUES (p_tenant_id, p_actor_id, p_actor_membership, p_accion, p_recurso_type,
            p_recurso_id, p_request_id, p_ip, p_user_agent, p_diff)
    RETURNING id INTO v_id;
    RETURN v_id;
END;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.prevent_audit_mutation()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'audit_events es append-only';
END;
$$;
-- +goose StatementEnd

DROP TRIGGER IF EXISTS audit_events_no_update ON audit_events;
CREATE TRIGGER audit_events_no_update BEFORE UPDATE ON audit_events
    FOR EACH ROW EXECUTE FUNCTION app.prevent_audit_mutation();
DROP TRIGGER IF EXISTS audit_events_no_delete ON audit_events;
CREATE TRIGGER audit_events_no_delete BEFORE DELETE ON audit_events
    FOR EACH ROW EXECUTE FUNCTION app.prevent_audit_mutation();
DROP TRIGGER IF EXISTS audit_events_no_truncate ON audit_events;
CREATE TRIGGER audit_events_no_truncate BEFORE TRUNCATE ON audit_events
    FOR EACH STATEMENT EXECUTE FUNCTION app.prevent_audit_mutation();

-- RLS: SELECT acotado por tenant (o al actor en eventos globales de identidad).
-- El INSERT va por la función SECURITY DEFINER (el rol app no tiene INSERT).
ALTER TABLE audit_events ENABLE ROW LEVEL SECURITY;
CREATE POLICY audit_events_visible ON audit_events FOR SELECT
    USING (tenant_id = app.current_tenant_id() OR actor_id = app.current_user_id());

GRANT SELECT ON audit_events TO consorcio_app;
GRANT EXECUTE ON FUNCTION app.record_audit_event(UUID, UUID, UUID, TEXT, TEXT, TEXT, TEXT, TEXT, TEXT, JSONB) TO consorcio_app;

-- +goose Down
REVOKE EXECUTE ON FUNCTION app.record_audit_event(UUID, UUID, UUID, TEXT, TEXT, TEXT, TEXT, TEXT, TEXT, JSONB) FROM consorcio_app;
REVOKE SELECT ON audit_events FROM consorcio_app;

DROP POLICY IF EXISTS audit_events_visible ON audit_events;
ALTER TABLE audit_events DISABLE ROW LEVEL SECURITY;

DROP TRIGGER IF EXISTS audit_events_no_truncate ON audit_events;
DROP TRIGGER IF EXISTS audit_events_no_delete ON audit_events;
DROP TRIGGER IF EXISTS audit_events_no_update ON audit_events;
DROP FUNCTION IF EXISTS app.prevent_audit_mutation();
DROP FUNCTION IF EXISTS app.record_audit_event(UUID, UUID, UUID, TEXT, TEXT, TEXT, TEXT, TEXT, TEXT, JSONB);
DROP TABLE IF EXISTS audit_events;
