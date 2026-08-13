-- name: RecordAuditEvent :one
SELECT app.record_audit_event(
    sqlc.arg(tenant_id)::UUID,
    sqlc.arg(actor_id)::UUID,
    sqlc.arg(actor_membership)::UUID,
    sqlc.arg(accion)::TEXT,
    sqlc.arg(recurso_type)::TEXT,
    sqlc.arg(recurso_id)::TEXT,
    sqlc.arg(request_id)::TEXT,
    sqlc.arg(ip)::TEXT,
    sqlc.arg(user_agent)::TEXT,
    sqlc.arg(diff)::JSONB
) AS id;

-- name: ListAuditEvents :many
SELECT id, tenant_id, actor_id, actor_membership, accion, recurso_type,
       recurso_id, request_id, ip, user_agent, diff, created_at
FROM audit_events
WHERE tenant_id = app.current_tenant_id()
  AND (sqlc.arg(accion)::TEXT = '' OR accion = sqlc.arg(accion)::TEXT)
  AND (sqlc.arg(recurso_type)::TEXT = '' OR recurso_type = sqlc.arg(recurso_type)::TEXT)
  AND (sqlc.arg(desde)::TIMESTAMPTZ IS NULL OR created_at >= sqlc.arg(desde)::TIMESTAMPTZ)
  AND (sqlc.arg(hasta)::TIMESTAMPTZ IS NULL OR created_at <= sqlc.arg(hasta)::TIMESTAMPTZ)
  AND (sqlc.arg(cursor_created)::TIMESTAMPTZ IS NULL
       OR (created_at, id) < (sqlc.arg(cursor_created)::TIMESTAMPTZ, sqlc.arg(cursor_id)::UUID))
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(page_limit)::INT;
