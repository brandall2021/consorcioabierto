-- name: ListProveedores :many
SELECT tenant_id, consorcio_id, id, cuit, razon_social, contacto_nombre, contacto_email, contacto_telefono, estado, created_at, updated_at
FROM proveedores
WHERE tenant_id = app.current_tenant_id()
  AND consorcio_id = sqlc.arg(consorcio_id)::UUID
  AND (sqlc.arg(q)::TEXT = '' OR razon_social ILIKE '%' || sqlc.arg(q)::TEXT || '%')
  AND (sqlc.arg(estado)::TEXT = '' OR estado = sqlc.arg(estado)::TEXT)
ORDER BY created_at DESC, id DESC;

-- name: GetProveedor :one
SELECT tenant_id, consorcio_id, id, cuit, razon_social, contacto_nombre, contacto_email, contacto_telefono, estado, created_at, updated_at
FROM proveedores
WHERE tenant_id = app.current_tenant_id()
  AND id = sqlc.arg(id)::UUID;

-- name: CreateProveedor :one
INSERT INTO proveedores (tenant_id, consorcio_id, cuit, razon_social, contacto_nombre, contacto_email, contacto_telefono)
VALUES (app.current_tenant_id(), sqlc.arg(consorcio_id)::UUID, sqlc.arg(cuit)::TEXT, sqlc.arg(razon_social)::TEXT,
        sqlc.narg(contacto_nombre)::TEXT, sqlc.narg(contacto_email)::TEXT, sqlc.narg(contacto_telefono)::TEXT)
RETURNING tenant_id, consorcio_id, id, cuit, razon_social, contacto_nombre, contacto_email, contacto_telefono, estado, created_at, updated_at;

-- name: UpdateProveedor :one
UPDATE proveedores
SET cuit = sqlc.arg(cuit)::TEXT,
    razon_social = sqlc.arg(razon_social)::TEXT,
    contacto_nombre = COALESCE(sqlc.narg(contacto_nombre)::TEXT, contacto_nombre),
    contacto_email = COALESCE(sqlc.narg(contacto_email)::TEXT, contacto_email),
    contacto_telefono = COALESCE(sqlc.narg(contacto_telefono)::TEXT, contacto_telefono),
    updated_at = now()
WHERE tenant_id = app.current_tenant_id()
  AND id = sqlc.arg(id)::UUID
RETURNING tenant_id, consorcio_id, id, cuit, razon_social, contacto_nombre, contacto_email, contacto_telefono, estado, created_at, updated_at;

-- name: UpdateProveedorEstado :one
UPDATE proveedores
SET estado = sqlc.arg(estado)::TEXT, updated_at = now()
WHERE tenant_id = app.current_tenant_id()
  AND id = sqlc.arg(id)::UUID
RETURNING tenant_id, consorcio_id, id, cuit, razon_social, contacto_nombre, contacto_email, contacto_telefono, estado, created_at, updated_at;