-- name: ListConsorcios :many
SELECT tenant_id, id, nombre, cuit, domicilio, tipo, estado, config, created_at, updated_at
FROM consorcios
WHERE tenant_id = app.current_tenant_id()
  AND (sqlc.arg(q)::TEXT = '' OR nombre ILIKE '%' || sqlc.arg(q)::TEXT || '%')
  AND (sqlc.arg(estado)::TEXT = '' OR estado = sqlc.arg(estado)::TEXT)
ORDER BY created_at DESC, id DESC;

-- name: GetConsorcio :one
SELECT tenant_id, id, nombre, cuit, domicilio, tipo, estado, config, created_at, updated_at
FROM consorcios
WHERE tenant_id = app.current_tenant_id()
  AND id = sqlc.arg(id)::UUID;

-- name: CreateConsorcio :one
INSERT INTO consorcios (tenant_id, nombre, cuit, domicilio, tipo)
VALUES (app.current_tenant_id(), sqlc.arg(nombre)::TEXT, sqlc.narg(cuit)::TEXT,
        sqlc.narg(domicilio)::TEXT, sqlc.arg(tipo)::TEXT)
RETURNING tenant_id, id, nombre, cuit, domicilio, tipo, estado, config, created_at, updated_at;

-- name: UpdateConsorcio :one
UPDATE consorcios
SET nombre = COALESCE(sqlc.narg(nombre)::TEXT, nombre),
    cuit = COALESCE(sqlc.narg(cuit)::TEXT, cuit),
    domicilio = COALESCE(sqlc.narg(domicilio)::TEXT, domicilio),
    tipo = COALESCE(sqlc.narg(tipo)::TEXT, tipo),
    estado = COALESCE(sqlc.narg(estado)::TEXT, estado),
    updated_at = now()
WHERE tenant_id = app.current_tenant_id()
  AND id = sqlc.arg(id)::UUID
RETURNING tenant_id, id, nombre, cuit, domicilio, tipo, estado, config, created_at, updated_at;
