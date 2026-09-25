-- name: ListComunicados :many
SELECT tenant_id, consorcio_id, id, titulo, cuerpo, destinatarios, estado, publicado_at, created_by, created_at, updated_at
FROM comunicados
WHERE tenant_id = app.current_tenant_id()
  AND consorcio_id = sqlc.arg(consorcio_id)::UUID
ORDER BY created_at DESC, id DESC;

-- name: GetComunicado :one
SELECT tenant_id, consorcio_id, id, titulo, cuerpo, destinatarios, estado, publicado_at, created_by, created_at, updated_at
FROM comunicados
WHERE tenant_id = app.current_tenant_id()
  AND id = sqlc.arg(id)::UUID;

-- name: CreateComunicado :one
INSERT INTO comunicados (tenant_id, consorcio_id, titulo, cuerpo, destinatarios, created_by)
VALUES (app.current_tenant_id(), sqlc.arg(consorcio_id)::UUID, sqlc.arg(titulo)::TEXT, sqlc.arg(cuerpo)::TEXT, sqlc.arg(destinatarios)::TEXT, sqlc.arg(created_by)::UUID)
RETURNING tenant_id, consorcio_id, id, titulo, cuerpo, destinatarios, estado, publicado_at, created_by, created_at, updated_at;

-- name: PublishComunicado :one
UPDATE comunicados
SET estado = 'publicado', publicado_at = COALESCE(publicado_at, now()), updated_at = now()
WHERE tenant_id = app.current_tenant_id()
  AND id = sqlc.arg(id)::UUID
  AND estado = 'borrador'
RETURNING tenant_id, consorcio_id, id, titulo, cuerpo, destinatarios, estado, publicado_at, created_by, created_at, updated_at;

-- name: ListComunicadoRecipients :many
SELECT DISTINCT p.email, p.nombre, up.unidad_id
FROM comunicados c
JOIN unidades u ON u.tenant_id = c.tenant_id AND u.consorcio_id = c.consorcio_id
JOIN unidad_personas up ON up.tenant_id = u.tenant_id AND up.unidad_id = u.id AND up.valid_to IS NULL
JOIN personas p ON p.tenant_id = up.tenant_id AND p.id = up.persona_id
WHERE c.tenant_id = app.current_tenant_id()
  AND c.id = sqlc.arg(id)::UUID
  AND p.email IS NOT NULL
ORDER BY p.email ASC;
