-- name: ListComunicados :many
SELECT tenant_id, id, consorcio_id, titulo, cuerpo, destinatarios_scope, destinatarios, estado, publicado_at, created_at, updated_at
FROM comunicados
WHERE tenant_id = app.current_tenant_id()
  AND consorcio_id = sqlc.arg(consorcio_id)::UUID
ORDER BY created_at DESC, id DESC;

-- name: GetComunicado :one
SELECT tenant_id, id, consorcio_id, titulo, cuerpo, destinatarios_scope, destinatarios, estado, publicado_at, created_at, updated_at
FROM comunicados
WHERE tenant_id = app.current_tenant_id()
  AND id = sqlc.arg(id)::UUID;

-- name: CreateComunicado :one
INSERT INTO comunicados (tenant_id, consorcio_id, titulo, cuerpo, destinatarios_scope, destinatarios)
VALUES (app.current_tenant_id(), sqlc.arg(consorcio_id)::UUID, sqlc.arg(titulo)::TEXT, sqlc.arg(cuerpo)::TEXT,
        sqlc.arg(destinatarios_scope)::TEXT, sqlc.arg(destinatarios)::BIGINT)
RETURNING tenant_id, id, consorcio_id, titulo, cuerpo, destinatarios_scope, destinatarios, estado, publicado_at, created_at, updated_at;

-- name: PublishComunicado :one
UPDATE comunicados
SET estado = 'publicado',
    publicado_at = COALESCE(publicado_at, now()),
    updated_at = now()
WHERE tenant_id = app.current_tenant_id()
  AND id = sqlc.arg(id)::UUID
RETURNING tenant_id, id, consorcio_id, titulo, cuerpo, destinatarios_scope, destinatarios, estado, publicado_at, created_at, updated_at;

-- name: InsertOutboxEvent :one
INSERT INTO outbox_events (tenant_id, correlation_id, event_type, payload)
VALUES (app.current_tenant_id(), sqlc.arg(correlation_id)::TEXT, sqlc.arg(event_type)::TEXT, sqlc.arg(payload)::JSONB)
RETURNING tenant_id, id, correlation_id, event_type, payload, estado, intentos, last_error, next_attempt_at, created_at, updated_at;

-- name: ClaimPendingOutboxEvents :many
SELECT tenant_id, id, correlation_id, event_type, payload, estado, intentos, last_error, next_attempt_at, created_at, updated_at
FROM app.claim_pending_outbox_events(sqlc.arg(limit)::INT);

-- name: MarkOutboxEventProcessed :exec
UPDATE outbox_events
SET estado = 'procesado',
    last_error = '',
    updated_at = now()
WHERE tenant_id = app.current_tenant_id()
  AND id = sqlc.arg(id)::UUID;

-- name: MarkOutboxEventFailed :exec
UPDATE outbox_events
SET estado = 'fallido',
    last_error = sqlc.arg(last_error)::TEXT,
    next_attempt_at = sqlc.arg(next_attempt_at)::TIMESTAMPTZ,
    updated_at = now()
WHERE tenant_id = app.current_tenant_id()
  AND id = sqlc.arg(id)::UUID;
