-- name: ListDestinatariosNotificacion :many
-- Destinatarios de una notificacion de consorcio: usuarios de plataforma con
-- una persona vinculada a alguna UF del consorcio y vinculo vigente hoy.
-- DISTINCT porque una persona puede tener varias UFs del mismo consorcio.
SELECT DISTINCT p.user_id
FROM unidades u
JOIN unidad_personas up
  ON up.tenant_id = u.tenant_id
 AND up.unidad_id = u.id
JOIN personas p
  ON p.tenant_id = up.tenant_id
 AND p.id = up.persona_id
WHERE u.tenant_id = app.current_tenant_id()
  AND u.consorcio_id = sqlc.arg(consorcio_id)::UUID
  AND p.user_id IS NOT NULL
  AND up.valid_from <= CURRENT_DATE
  AND (up.valid_to IS NULL OR up.valid_to >= CURRENT_DATE)
ORDER BY p.user_id;

-- name: InsertNotificacion :one
INSERT INTO notificaciones (
    tenant_id, user_id, tipo, titulo, cuerpo, recurso_type, recurso_id
) VALUES (
    app.current_tenant_id(),
    sqlc.arg(user_id)::UUID,
    sqlc.arg(tipo)::TEXT,
    sqlc.arg(titulo)::TEXT,
    sqlc.arg(cuerpo)::TEXT,
    sqlc.arg(recurso_type)::TEXT,
    sqlc.narg(recurso_id)::UUID
)
ON CONFLICT DO NOTHING
RETURNING *;

-- name: ListNotificacionesForCurrentUser :many
SELECT *
FROM notificaciones
WHERE tenant_id = app.current_tenant_id()
  AND user_id = app.current_user_id()
  AND (sqlc.arg(solo_no_leidas)::BOOLEAN = FALSE OR leida_at IS NULL)
ORDER BY created_at DESC
LIMIT sqlc.arg(limite);

-- name: MarcarNotificacionLeida :one
UPDATE notificaciones
SET leida_at = now()
WHERE tenant_id = app.current_tenant_id()
  AND user_id = app.current_user_id()
  AND id = sqlc.arg(id)::UUID
  AND leida_at IS NULL
RETURNING *;

-- name: CountNotificacionesNoLeidas :one
SELECT count(*)
FROM notificaciones
WHERE tenant_id = app.current_tenant_id()
  AND user_id = app.current_user_id()
  AND leida_at IS NULL;
