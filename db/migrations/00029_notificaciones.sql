-- +goose Up
-- Notificaciones in-app del consorcista. Cierra el flujo que la spec pide al
-- publicar: "el worker genera documentos y envia notificaciones" (6.2) y
-- "publicada: documentos disponibles y notificaciones encoladas" (5.2).
--
-- Quien las crea es el worker del outbox, no el consorcista, asi que la policy
-- de INSERT acota por tenant (mismo criterio que comunicados y outbox_events).
-- La lectura si es por usuario: un consorcista solo ve las suyas.

CREATE TABLE IF NOT EXISTS notificaciones (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tipo TEXT NOT NULL,
    titulo TEXT NOT NULL,
    cuerpo TEXT NOT NULL DEFAULT '',
    recurso_type TEXT NOT NULL DEFAULT '',
    recurso_id UUID,
    leida_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, id),
    CONSTRAINT notificaciones_tipo_check CHECK (tipo IN ('comunicado', 'liquidacion'))
);

-- El worker reintenta los eventos fallidos: sin este indice una notificacion
-- duplicada llegaria al usuario cada vez que el reintento corre.
CREATE UNIQUE INDEX IF NOT EXISTS notificaciones_destinatario_recurso_unique
    ON notificaciones(tenant_id, user_id, tipo, recurso_id)
    WHERE recurso_id IS NOT NULL;

-- Indice del portal: notificaciones del usuario, las no leidas primero.
CREATE INDEX IF NOT EXISTS notificaciones_user_pendientes
    ON notificaciones(tenant_id, user_id, created_at DESC)
    WHERE leida_at IS NULL;

ALTER TABLE notificaciones ENABLE ROW LEVEL SECURITY;
CREATE POLICY notificaciones_visible ON notificaciones FOR SELECT
    USING (tenant_id = app.current_tenant_id() AND user_id = app.current_user_id());
CREATE POLICY notificaciones_insert ON notificaciones FOR INSERT
    WITH CHECK (tenant_id = app.current_tenant_id());
CREATE POLICY notificaciones_update ON notificaciones FOR UPDATE
    USING (tenant_id = app.current_tenant_id() AND user_id = app.current_user_id());

GRANT SELECT, INSERT, UPDATE ON notificaciones TO consorcio_app;

-- +goose Down
REVOKE SELECT, INSERT, UPDATE ON notificaciones FROM consorcio_app;
DROP INDEX IF EXISTS notificaciones_user_pendientes;
DROP INDEX IF EXISTS notificaciones_destinatario_recurso_unique;
DROP TABLE IF EXISTS notificaciones;
