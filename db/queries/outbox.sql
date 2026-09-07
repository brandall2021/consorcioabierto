-- name: InsertOutboxEvent :one
INSERT INTO outbox_events (tenant_id, correlation_id, event_type, payload)
VALUES (app.current_tenant_id(), sqlc.arg('correlation_id')::UUID, sqlc.arg('event_type')::TEXT,
        sqlc.arg('payload')::JSONB)
RETURNING tenant_id, id, correlation_id, event_type, payload, estado, intentos, next_attempt_at, created_at;

-- name: GetPendingOutboxEvents :many
SELECT tenant_id, id, correlation_id, event_type, payload, estado, intentos, next_attempt_at, created_at
FROM outbox_events
WHERE estado = 'pendiente' AND next_attempt_at <= now()
ORDER BY created_at ASC
LIMIT sqlc.arg('batch_size')::INT;

-- name: MarkOutboxProcessed :exec
UPDATE outbox_events SET estado = 'procesado', intentos = intentos + 1
WHERE tenant_id = app.current_tenant_id() AND id = sqlc.arg('id')::UUID;

-- name: MarkOutboxFailed :exec
UPDATE outbox_events
SET estado = 'fallido', intentos = intentos + 1, next_attempt_at = now() + INTERVAL '1 minute' * power(2, intentos)
WHERE tenant_id = app.current_tenant_id() AND id = sqlc.arg('id')::UUID;