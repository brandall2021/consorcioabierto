-- name: CreateLiquidacion :one
INSERT INTO liquidaciones (tenant_id, consorcio_id, periodo, vencimiento_1, vencimiento_2)
VALUES (app.current_tenant_id(), sqlc.arg('consorcio_id')::UUID, sqlc.arg('periodo')::TEXT,
        sqlc.arg('vencimiento_1')::DATE, sqlc.narg('vencimiento_2')::DATE)
ON CONFLICT (tenant_id, consorcio_id, periodo)
    WHERE estado NOT IN ('anulada', 'cerrada') DO NOTHING
RETURNING tenant_id, consorcio_id, id, periodo, vencimiento_1, vencimiento_2, estado, version,
          total_gastos_cents, total_distribuido_cents, unidades_alcanzadas, created_at, updated_at;

-- name: GetLiquidacion :one
SELECT tenant_id, consorcio_id, id, periodo, vencimiento_1, vencimiento_2, estado, version,
       total_gastos_cents, total_distribuido_cents, unidades_alcanzadas, created_at, updated_at
FROM liquidaciones
WHERE tenant_id = app.current_tenant_id()
  AND consorcio_id = sqlc.arg('consorcio_id')::UUID
  AND id = sqlc.arg('id')::UUID;

-- name: ListLiquidaciones :many
SELECT tenant_id, consorcio_id, id, periodo, vencimiento_1, vencimiento_2, estado, version,
       total_gastos_cents, total_distribuido_cents, unidades_alcanzadas, created_at, updated_at
FROM liquidaciones
WHERE tenant_id = app.current_tenant_id()
  AND consorcio_id = sqlc.arg('consorcio_id')::UUID
  AND (sqlc.arg('periodo')::TEXT = '' OR periodo = sqlc.arg('periodo')::TEXT)
  AND (sqlc.arg('estado')::TEXT = '' OR estado = sqlc.arg('estado')::TEXT)
ORDER BY periodo DESC, created_at DESC, id DESC;

-- name: UpdateLiquidacionVencimientos :exec
UPDATE liquidaciones
SET vencimiento_1 = sqlc.arg('vencimiento_1')::DATE,
    vencimiento_2 = sqlc.narg('vencimiento_2')::DATE,
    updated_at = now()
WHERE tenant_id = app.current_tenant_id()
  AND consorcio_id = sqlc.arg('consorcio_id')::UUID
  AND id = sqlc.arg('id')::UUID
  AND estado = 'borrador'
  AND version = sqlc.arg('expected_version')::INT;

-- name: TransitionLiquidacion :execrows
UPDATE liquidaciones
SET estado = sqlc.arg('nuevo_estado')::TEXT,
    version = version + 1,
    updated_at = now()
WHERE tenant_id = app.current_tenant_id()
  AND consorcio_id = sqlc.arg('consorcio_id')::UUID
  AND id = sqlc.arg('id')::UUID
  AND estado = sqlc.arg('estado_actual')::TEXT
  AND version = sqlc.arg('expected_version')::INT;

-- name: UpdateLiquidacionCalculo :exec
UPDATE liquidaciones
SET total_gastos_cents = sqlc.arg('total_gastos_cents')::BIGINT,
    total_distribuido_cents = sqlc.arg('total_distribuido_cents')::BIGINT,
    unidades_alcanzadas = sqlc.arg('unidades_alcanzadas')::INT,
    updated_at = now()
WHERE tenant_id = app.current_tenant_id()
  AND id = sqlc.arg('id')::UUID;

-- name: InsertLiquidacionGasto :exec
INSERT INTO liquidacion_gastos (tenant_id, liquidacion_id, gasto_id, concepto_id, importe_cents)
VALUES (app.current_tenant_id(), sqlc.arg('liquidacion_id')::UUID, sqlc.arg('gasto_id')::UUID,
        sqlc.arg('concepto_id')::UUID, sqlc.arg('importe_cents')::BIGINT);

-- name: InsertLiquidacionItem :exec
INSERT INTO liquidacion_items (tenant_id, liquidacion_id, concepto_id, regla_aplicada, importe_cents)
VALUES (app.current_tenant_id(), sqlc.arg('liquidacion_id')::UUID, sqlc.arg('concepto_id')::UUID,
        sqlc.arg('regla_aplicada')::TEXT, sqlc.arg('importe_cents')::BIGINT);

-- name: InsertLiquidacionUnidad :exec
INSERT INTO liquidacion_unidades (tenant_id, liquidacion_id, unidad_id, codigo, coeficiente, total_cents)
VALUES (app.current_tenant_id(), sqlc.arg('liquidacion_id')::UUID, sqlc.arg('unidad_id')::UUID,
        sqlc.arg('codigo')::TEXT, sqlc.arg('coeficiente')::NUMERIC, sqlc.arg('total_cents')::BIGINT);

-- name: InsertLiquidacionUnidadItem :exec
INSERT INTO liquidacion_unidad_items (tenant_id, liquidacion_id, liquidacion_unidad_id, concepto_id, importe_cents)
VALUES (app.current_tenant_id(), sqlc.arg('liquidacion_id')::UUID, sqlc.arg('liquidacion_unidad_id')::UUID,
        sqlc.arg('concepto_id')::UUID, sqlc.arg('importe_cents')::BIGINT);

-- name: GetLiquidacionGastos :many
SELECT tenant_id, liquidacion_id, gasto_id, concepto_id, importe_cents
FROM liquidacion_gastos
WHERE tenant_id = app.current_tenant_id() AND liquidacion_id = sqlc.arg('liquidacion_id')::UUID;

-- name: GetLiquidacionItems :many
SELECT tenant_id, liquidacion_id, concepto_id, regla_aplicada, importe_cents
FROM liquidacion_items
WHERE tenant_id = app.current_tenant_id() AND liquidacion_id = sqlc.arg('liquidacion_id')::UUID;

-- name: GetLiquidacionUnidades :many
SELECT tenant_id, liquidacion_id, unidad_id, codigo, coeficiente, total_cents
FROM liquidacion_unidades
WHERE tenant_id = app.current_tenant_id() AND liquidacion_id = sqlc.arg('liquidacion_id')::UUID;

-- name: GetLiquidacionUnidadItems :many
SELECT tenant_id, liquidacion_id, liquidacion_unidad_id, concepto_id, importe_cents
FROM liquidacion_unidad_items
WHERE tenant_id = app.current_tenant_id() AND liquidacion_id = sqlc.arg('liquidacion_id')::UUID;

-- name: AnularLiquidacion :execrows
UPDATE liquidaciones
SET estado = 'anulada', version = version + 1, updated_at = now()
WHERE tenant_id = app.current_tenant_id()
  AND id = sqlc.arg('id')::UUID
  AND estado = sqlc.arg('estado_actual')::TEXT
  AND version = sqlc.arg('expected_version')::INT;

-- name: DeleteLiquidacionGastos :exec
DELETE FROM liquidacion_gastos
WHERE tenant_id = app.current_tenant_id()
  AND liquidacion_id = sqlc.arg('liquidacion_id')::UUID;

-- name: DeleteLiquidacionItems :exec
DELETE FROM liquidacion_items
WHERE tenant_id = app.current_tenant_id()
  AND liquidacion_id = sqlc.arg('liquidacion_id')::UUID;

-- name: DeleteLiquidacionUnidades :exec
DELETE FROM liquidacion_unidades
WHERE tenant_id = app.current_tenant_id()
  AND liquidacion_id = sqlc.arg('liquidacion_id')::UUID;
