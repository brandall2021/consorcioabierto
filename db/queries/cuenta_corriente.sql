-- name: InsertCharge :one
INSERT INTO charges (tenant_id, unidad_id, liquidacion_id, concepto, due_date, total_cents, saldo_cents)
VALUES (app.current_tenant_id(), sqlc.arg('unidad_id')::UUID, sqlc.narg('liquidacion_id')::UUID,
        sqlc.arg('concepto')::TEXT, sqlc.arg('due_date')::DATE,
        sqlc.arg('total_cents')::BIGINT, sqlc.arg('saldo_cents')::BIGINT)
RETURNING tenant_id, unidad_id, id, liquidacion_id, concepto, due_date, total_cents, saldo_cents, created_at;

-- name: GetCharge :one
SELECT tenant_id, unidad_id, id, liquidacion_id, concepto, due_date, total_cents, saldo_cents, created_at
FROM charges
WHERE tenant_id = app.current_tenant_id() AND id = sqlc.arg('id')::UUID;

-- name: UpdateChargeSaldo :exec
UPDATE charges SET saldo_cents = sqlc.arg('saldo_cents')::BIGINT
WHERE tenant_id = app.current_tenant_id() AND id = sqlc.arg('id')::UUID;

-- name: InsertAccountEntry :one
INSERT INTO account_entries (tenant_id, unidad_id, tipo, fecha_efectiva, debit_cents, credit_cents, currency, referencia, reversa_de_id, charge_id)
VALUES (app.current_tenant_id(), sqlc.arg('unidad_id')::UUID, sqlc.arg('tipo')::TEXT,
        sqlc.arg('fecha_efectiva')::DATE, sqlc.arg('debit_cents')::BIGINT, sqlc.arg('credit_cents')::BIGINT,
        sqlc.arg('currency')::TEXT, sqlc.narg('referencia')::TEXT,
        sqlc.narg('reversa_de_id')::UUID, sqlc.narg('charge_id')::UUID)
RETURNING tenant_id, unidad_id, id, tipo, fecha_efectiva, debit_cents, credit_cents, currency, referencia, reversa_de_id, charge_id, created_at;

-- name: GetEntriesByUnidad :many
SELECT tenant_id, unidad_id, id, tipo, fecha_efectiva, debit_cents, credit_cents, currency, referencia, reversa_de_id, charge_id, created_at
FROM account_entries
WHERE tenant_id = app.current_tenant_id() AND unidad_id = sqlc.arg('unidad_id')::UUID
ORDER BY created_at ASC, id ASC;

-- name: GetChargesByLiquidacion :many
SELECT tenant_id, unidad_id, id, liquidacion_id, concepto, due_date, total_cents, saldo_cents, created_at
FROM charges
WHERE tenant_id = app.current_tenant_id() AND liquidacion_id = sqlc.arg('liquidacion_id')::UUID;

-- name: ListMorosidadByConsorcio :many
SELECT u.id AS unidad_id,
       u.codigo AS unidad_codigo,
       SUM(c.saldo_cents)::BIGINT AS saldo_vencido_cents,
       COUNT(*)::BIGINT AS cantidad_cargos,
       MIN(c.due_date) AS vencido_desde
FROM charges c
JOIN unidades u ON u.tenant_id = c.tenant_id AND u.id = c.unidad_id
WHERE c.tenant_id = app.current_tenant_id()
  AND u.consorcio_id = sqlc.arg('consorcio_id')::UUID
  AND c.due_date < CURRENT_DATE
  AND c.saldo_cents > 0
GROUP BY u.id, u.codigo
ORDER BY saldo_vencido_cents DESC, u.codigo ASC;
