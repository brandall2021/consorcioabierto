-- name: ListCobranzas :many
SELECT tenant_id, unidad_id, id, fecha, canal, importe_cents, referencia, estado, idem_key, motivo_rechazo, created_by, created_at, psp_provider, psp_preference_id, psp_checkout_url
FROM payments
WHERE tenant_id = app.current_tenant_id()
  AND unidad_id IN (
      SELECT id
      FROM unidades
      WHERE tenant_id = app.current_tenant_id()
        AND consorcio_id = sqlc.arg(consorcio_id)::UUID
  )
ORDER BY created_at DESC, id DESC;

-- name: CreateCobranza :one
INSERT INTO payments (tenant_id, unidad_id, fecha, canal, importe_cents, referencia, estado, idem_key, created_by)
VALUES (app.current_tenant_id(), sqlc.arg(unidad_id)::UUID, sqlc.arg(fecha)::DATE,
        sqlc.arg(canal)::TEXT, sqlc.arg(importe_cents)::BIGINT, sqlc.narg(referencia)::TEXT,
        sqlc.arg(estado)::TEXT, sqlc.arg(idem_key)::TEXT, sqlc.arg(created_by)::UUID)
RETURNING tenant_id, unidad_id, id, fecha, canal, importe_cents, referencia, estado, idem_key, motivo_rechazo, created_by, created_at, psp_provider, psp_preference_id, psp_checkout_url;

-- name: UpdateCobranzaMercadoPago :exec
UPDATE payments
SET psp_provider = sqlc.narg(psp_provider)::TEXT,
    psp_preference_id = sqlc.narg(psp_preference_id)::TEXT,
    psp_checkout_url = sqlc.narg(psp_checkout_url)::TEXT
WHERE tenant_id = app.current_tenant_id()
  AND id = sqlc.arg(id)::UUID;

-- name: CreatePspIntent :one
INSERT INTO psp_intents (tenant_id, payment_id, provider, preference_id, status)
VALUES (app.current_tenant_id(), sqlc.arg(payment_id)::UUID, sqlc.arg(provider)::TEXT, sqlc.arg(preference_id)::TEXT, sqlc.arg(status)::TEXT)
ON CONFLICT (tenant_id, payment_id) DO UPDATE
SET provider = EXCLUDED.provider,
    preference_id = EXCLUDED.preference_id,
    status = EXCLUDED.status,
    updated_at = now()
RETURNING tenant_id, id, payment_id, provider, preference_id, status, created_at, updated_at;

-- name: GetPspIntentByPreferenceID :one
SELECT tenant_id, id, payment_id, provider, preference_id, status, created_at, updated_at
FROM psp_intents
WHERE tenant_id = app.current_tenant_id()
  AND preference_id = sqlc.arg(preference_id)::TEXT;

-- name: UpdatePspIntentStatus :exec
UPDATE psp_intents
SET status = sqlc.arg(status)::TEXT,
    updated_at = now()
WHERE tenant_id = app.current_tenant_id()
  AND preference_id = sqlc.arg(preference_id)::TEXT;

-- name: GetCobranza :one
SELECT tenant_id, unidad_id, id, fecha, canal, importe_cents, referencia, estado, idem_key, motivo_rechazo, created_by, created_at, psp_provider, psp_preference_id, psp_checkout_url
FROM payments
WHERE tenant_id = app.current_tenant_id()
  AND id = sqlc.arg(id)::UUID;

-- name: UpdateCobranzaEstado :exec
UPDATE payments
SET estado = sqlc.arg(estado)::TEXT,
    motivo_rechazo = sqlc.narg(motivo_rechazo)::TEXT
WHERE tenant_id = app.current_tenant_id()
  AND id = sqlc.arg(id)::UUID;

-- name: ListOpenChargesByUnidad :many
SELECT tenant_id, unidad_id, id, liquidacion_id, concepto, due_date, total_cents, saldo_cents, created_at
FROM charges
WHERE tenant_id = app.current_tenant_id()
  AND unidad_id = sqlc.arg(unidad_id)::UUID
  AND saldo_cents > 0
  AND due_date <= sqlc.arg(fecha_corte)::DATE
ORDER BY due_date ASC, created_at ASC, id ASC;

-- name: InsertPaymentAllocation :one
INSERT INTO payment_allocations (tenant_id, payment_id, charge_id, amount_cents, created_by)
VALUES (app.current_tenant_id(), sqlc.arg(payment_id)::UUID, sqlc.arg(charge_id)::UUID,
        sqlc.arg(amount_cents)::BIGINT, sqlc.arg(created_by)::UUID)
RETURNING tenant_id, payment_id, charge_id, amount_cents, created_by, created_at;

-- name: ListPaymentAllocationsByPayment :many
SELECT tenant_id, payment_id, charge_id, amount_cents, created_by, created_at
FROM payment_allocations
WHERE tenant_id = app.current_tenant_id()
  AND payment_id = sqlc.arg(payment_id)::UUID
ORDER BY created_at ASC, charge_id ASC;

-- name: DeletePaymentAllocationsByPayment :execrows
DELETE FROM payment_allocations
WHERE tenant_id = app.current_tenant_id()
  AND payment_id = sqlc.arg(payment_id)::UUID;
