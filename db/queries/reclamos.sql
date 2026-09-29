-- name: ListReclamos :many
SELECT tenant_id, id, consorcio_id, unidad_id, categoria, texto, estado, responsable_id, sla_due_at, created_by, created_at, updated_at
FROM reclamos
WHERE tenant_id = app.current_tenant_id()
  AND consorcio_id = $1::UUID
  AND (estado = $2::TEXT OR $2 = '')
ORDER BY created_at DESC, id DESC;

-- name: GetReclamo :one
SELECT tenant_id, id, consorcio_id, unidad_id, categoria, texto, estado, responsable_id, sla_due_at, created_by, created_at, updated_at
FROM reclamos
WHERE tenant_id = app.current_tenant_id()
  AND id = $1::UUID;

-- name: CreateReclamo :one
INSERT INTO reclamos (tenant_id, consorcio_id, unidad_id, categoria, texto, sla_due_at)
VALUES (app.current_tenant_id(), $1::UUID, $2::UUID, $3::TEXT, $4::TEXT, $5::TIMESTAMPTZ)
RETURNING tenant_id, id, consorcio_id, unidad_id, categoria, texto, estado, responsable_id, sla_due_at, created_by, created_at, updated_at;

-- name: UpdateReclamoEstado :one
UPDATE reclamos
SET estado = $2::TEXT,
    responsable_id = $3::UUID,
    updated_at = now()
WHERE tenant_id = app.current_tenant_id()
  AND id = $1::UUID
RETURNING tenant_id, id, consorcio_id, unidad_id, categoria, texto, estado, responsable_id, sla_due_at, created_by, created_at, updated_at;

-- name: ListReclamoMensajes :many
SELECT tenant_id, id, reclamo_id, texto, adjunto_id, created_by, created_at
FROM reclamo_mensajes
WHERE tenant_id = app.current_tenant_id()
  AND reclamo_id = $1::UUID
ORDER BY created_at ASC, id ASC;

-- name: InsertReclamoMensaje :one
INSERT INTO reclamo_mensajes (tenant_id, reclamo_id, texto, adjunto_id)
VALUES (app.current_tenant_id(), $1::UUID, $2::TEXT, $3::UUID)
RETURNING tenant_id, id, reclamo_id, texto, adjunto_id, created_by, created_at;

-- name: ListReclamoTransiciones :many
SELECT tenant_id, id, reclamo_id, accion, motivo, from_estado, to_estado, responsable_id, created_by, created_at
FROM reclamo_transiciones
WHERE tenant_id = app.current_tenant_id()
  AND reclamo_id = $1::UUID
ORDER BY created_at ASC, id ASC;

-- name: InsertReclamoTransicion :one
INSERT INTO reclamo_transiciones (tenant_id, reclamo_id, accion, motivo, from_estado, to_estado, responsable_id)
VALUES (app.current_tenant_id(), $1::UUID, $2::TEXT, $3::TEXT, $4::TEXT, $5::TEXT, $6::UUID)
RETURNING tenant_id, id, reclamo_id, accion, motivo, from_estado, to_estado, responsable_id, created_by, created_at;