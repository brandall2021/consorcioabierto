-- name: ListConceptos :many
SELECT tenant_id, consorcio_id, id, nombre, categoria, regla, created_at, updated_at
FROM conceptos_expensa
WHERE tenant_id = app.current_tenant_id()
  AND (consorcio_id = sqlc.arg('consorcio_id')::UUID OR consorcio_id IS NULL)
  AND (sqlc.arg('q')::TEXT = '' OR nombre ILIKE '%' || sqlc.arg('q')::TEXT || '%')
ORDER BY nombre ASC, id DESC;

-- name: GetConcepto :one
SELECT tenant_id, consorcio_id, id, nombre, categoria, regla, created_at, updated_at
FROM conceptos_expensa
WHERE tenant_id = app.current_tenant_id()
  AND id = sqlc.arg(id)::UUID;

-- name: CreateConcepto :one
INSERT INTO conceptos_expensa (tenant_id, consorcio_id, nombre, categoria, regla)
VALUES (app.current_tenant_id(), sqlc.arg('consorcio_id')::UUID, sqlc.arg('nombre')::TEXT, sqlc.arg('categoria')::TEXT, 'coeficiente')
RETURNING tenant_id, consorcio_id, id, nombre, categoria, regla, created_at, updated_at;
