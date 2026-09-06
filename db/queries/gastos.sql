-- name: ListGastos :many
SELECT tenant_id, consorcio_id, id, concepto_id, proveedor_id, comprobante, importe_cents, fecha, estado, documento_id, created_at, updated_at
FROM gastos
WHERE tenant_id = app.current_tenant_id()
  AND consorcio_id = sqlc.arg('consorcio_id')::UUID
  AND (sqlc.arg('mes')::TEXT = '' OR to_char(fecha, 'YYYY-MM') = sqlc.arg('mes')::TEXT)
  AND proveedor_id = COALESCE(sqlc.narg('proveedor_id')::UUID, proveedor_id)
ORDER BY fecha DESC, created_at DESC, id DESC;

-- name: GetGasto :one
SELECT tenant_id, consorcio_id, id, concepto_id, proveedor_id, comprobante, importe_cents, fecha, estado, documento_id, created_at, updated_at
FROM gastos
WHERE tenant_id = app.current_tenant_id()
  AND consorcio_id = sqlc.arg(consorcio_id)::UUID
  AND id = sqlc.arg(id)::UUID;

-- name: CreateGasto :one
INSERT INTO gastos (tenant_id, consorcio_id, concepto_id, proveedor_id, comprobante, importe_cents, fecha, documento_id)
VALUES (app.current_tenant_id(), sqlc.arg('consorcio_id')::UUID, sqlc.arg('concepto_id')::UUID,
        sqlc.narg('proveedor_id')::UUID, sqlc.narg('comprobante')::TEXT, sqlc.arg('importe_cents')::BIGINT,
        sqlc.arg('fecha')::DATE, sqlc.narg('documento_id')::UUID)
RETURNING tenant_id, consorcio_id, id, concepto_id, proveedor_id, comprobante, importe_cents, fecha, estado, documento_id, created_at, updated_at;
