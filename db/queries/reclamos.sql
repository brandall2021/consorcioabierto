-- name: ListReclamos :many
SELECT tenant_id, consorcio_id, unidad_id, id, categoria, texto, estado, responsable_id, created_by, created_at, updated_at
FROM reclamos
WHERE tenant_id = app.current_tenant_id()
  AND consorcio_id = sqlc.arg(consorcio_id)::UUID
ORDER BY created_at DESC, id DESC;

-- name: GetReclamo :one
SELECT tenant_id, consorcio_id, unidad_id, id, categoria, texto, estado, responsable_id, created_by, created_at, updated_at
FROM reclamos
WHERE tenant_id = app.current_tenant_id()
  AND id = sqlc.arg(id)::UUID;

-- name: CreateReclamo :one
INSERT INTO reclamos (tenant_id, consorcio_id, unidad_id, categoria, texto, created_by)
VALUES (app.current_tenant_id(), sqlc.arg(consorcio_id)::UUID, sqlc.arg(unidad_id)::UUID, sqlc.arg(categoria)::TEXT, sqlc.arg(texto)::TEXT, sqlc.arg(created_by)::UUID)
RETURNING tenant_id, consorcio_id, unidad_id, id, categoria, texto, estado, responsable_id, created_by, created_at, updated_at;

-- name: UpdateReclamo :one
UPDATE reclamos
SET estado = COALESCE(sqlc.narg(estado)::TEXT, estado),
    responsable_id = COALESCE(sqlc.narg(responsable_id)::UUID, responsable_id),
    updated_at = now()
WHERE tenant_id = app.current_tenant_id()
  AND id = sqlc.arg(id)::UUID
RETURNING tenant_id, consorcio_id, unidad_id, id, categoria, texto, estado, responsable_id, created_by, created_at, updated_at;

-- name: ListReclamoMensajes :many
SELECT tenant_id, reclamo_id, id, texto, adjunto_id, created_by, created_at
FROM reclamo_mensajes
WHERE tenant_id = app.current_tenant_id()
  AND reclamo_id = sqlc.arg(reclamo_id)::UUID
ORDER BY created_at ASC, id ASC;

-- name: CreateReclamoMensaje :one
INSERT INTO reclamo_mensajes (tenant_id, reclamo_id, texto, adjunto_id, created_by)
VALUES (app.current_tenant_id(), sqlc.arg(reclamo_id)::UUID, sqlc.arg(texto)::TEXT, sqlc.narg(adjunto_id)::UUID, sqlc.arg(created_by)::UUID)
RETURNING tenant_id, reclamo_id, id, texto, adjunto_id, created_by, created_at;

-- name: ListReclamoTransiciones :many
SELECT tenant_id, reclamo_id, id, accion, motivo, responsable_id, created_by, created_at
FROM reclamo_transiciones
WHERE tenant_id = app.current_tenant_id()
  AND reclamo_id = sqlc.arg(reclamo_id)::UUID
ORDER BY created_at ASC, id ASC;

-- name: CreateReclamoTransicion :one
INSERT INTO reclamo_transiciones (tenant_id, reclamo_id, accion, motivo, responsable_id, created_by)
VALUES (app.current_tenant_id(), sqlc.arg(reclamo_id)::UUID, sqlc.arg(accion)::TEXT, sqlc.narg(motivo)::TEXT, sqlc.narg(responsable_id)::UUID, sqlc.arg(created_by)::UUID)
RETURNING tenant_id, reclamo_id, id, accion, motivo, responsable_id, created_by, created_at;
