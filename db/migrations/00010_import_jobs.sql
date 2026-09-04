-- +goose Up
-- Importación CSV de UFs (H2.3, §6.1). El job valida el archivo sin escribir
-- nada (preview), guarda filas y errores por fila, y el confirm aplica el modo
-- explícito (crear|actualizar|crear_y_actualizar) en una transacción.

CREATE TABLE IF NOT EXISTS import_jobs (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    consorcio_id UUID NOT NULL,
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    estado TEXT NOT NULL DEFAULT 'validando',
    modo TEXT NOT NULL,
    plantilla_version TEXT NOT NULL DEFAULT 'ufs-v1',
    total_filas INTEGER NOT NULL DEFAULT 0,
    filas JSONB NOT NULL DEFAULT '[]'::jsonb,
    errores JSONB NOT NULL DEFAULT '[]'::jsonb,
    creados INTEGER NOT NULL DEFAULT 0,
    actualizados INTEGER NOT NULL DEFAULT 0,
    rechazados INTEGER NOT NULL DEFAULT 0,
    archivo_errores_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, id),
    CONSTRAINT import_jobs_consorcio_fkey FOREIGN KEY (tenant_id, consorcio_id)
        REFERENCES consorcios(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT import_jobs_estado_check CHECK (estado IN ('validando','listo','confirmando','procesado','fallido')),
    CONSTRAINT import_jobs_modo_check CHECK (modo IN ('crear','actualizar','crear_y_actualizar'))
);

ALTER TABLE import_jobs ENABLE ROW LEVEL SECURITY;
CREATE POLICY import_jobs_visible ON import_jobs FOR SELECT USING (tenant_id = app.current_tenant_id());
CREATE POLICY import_jobs_insert ON import_jobs FOR INSERT WITH CHECK (tenant_id = app.current_tenant_id());
CREATE POLICY import_jobs_update ON import_jobs FOR UPDATE USING (tenant_id = app.current_tenant_id());

GRANT SELECT, INSERT, UPDATE ON import_jobs TO consorcio_app;

-- El confirm guarda el resultado idempotente en idempotency_keys (ADR-0004):
-- falta la política de UPDATE, necesaria para persistir response_json.
CREATE POLICY idempotency_update ON idempotency_keys FOR UPDATE USING (tenant_id = app.current_tenant_id());

-- +goose Down
DROP POLICY IF EXISTS idempotency_update ON idempotency_keys;

REVOKE SELECT, INSERT, UPDATE ON import_jobs FROM consorcio_app;

DROP POLICY IF EXISTS import_jobs_update ON import_jobs;
DROP POLICY IF EXISTS import_jobs_insert ON import_jobs;
DROP POLICY IF EXISTS import_jobs_visible ON import_jobs;

DROP TABLE IF EXISTS import_jobs;