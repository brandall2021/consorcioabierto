-- name: InsertIdempotencyKey :execrows
INSERT INTO idempotency_keys (tenant_id, idempotency_key, scope, request_hash, response_json)
VALUES (app.current_tenant_id(), sqlc.arg(idempotency_key)::TEXT, sqlc.arg(scope)::TEXT,
        sqlc.arg(request_hash)::TEXT, sqlc.arg(response_json)::JSONB)
ON CONFLICT (tenant_id, scope, idempotency_key) DO NOTHING;

-- name: GetIdempotencyKey :one
SELECT id, tenant_id, idempotency_key, scope, request_hash, response_json, created_at
FROM idempotency_keys
WHERE tenant_id = app.current_tenant_id()
  AND scope = sqlc.arg(scope)::TEXT
  AND idempotency_key = sqlc.arg(idempotency_key)::TEXT;

-- name: UpdateIdempotencyKey :exec
UPDATE idempotency_keys
SET response_json = sqlc.arg(response_json)::JSONB
WHERE tenant_id = app.current_tenant_id()
  AND scope = sqlc.arg(scope)::TEXT
  AND idempotency_key = sqlc.arg(idempotency_key)::TEXT;