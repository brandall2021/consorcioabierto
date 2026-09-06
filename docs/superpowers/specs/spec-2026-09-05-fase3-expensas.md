# Spec — Fase 3 (Expensas)

Fecha: 2026-09-05
Estado: Aprobada por el owner
Origen: `consorcioabierto-especificacion-mejorada-para-ia.md` (§5.1, §5.2, §5.4, §6.2, §7.2, §8.3, §15), ADR-0004, ADR-0006, `docs/roadmap.md` (H3.1–H3.7).

## Decisiones del owner (§15) documentadas en esta fase

| Decisión | Valor | Registro |
|---|---|---|
| Conceptos de expensa | Tabla `conceptos_expensa` con regla de distribución (§5.1) | Esta spec |
| `vencimiento_2` | Solo fecha; sin interés en MVP | Esta spec |
| Formato de liquidación | PDF simple generado por el backend | Esta spec |
| Redondeo | Mayor resto + tie-break por código UF | ADR-0006 |
| Imputación de cobros | FIFO + saldo a favor | ADR-0007 (Fase 4) |

## Arquitectura (Enfoque A — dominio puro, módulos separados)

- `internal/expensas/`: dominio de liquidaciones, gastos y concepto de expensa.
- `internal/cuenta_corriente/`: libro de movimientos (`account_entries`) y cargos (`charges`). Independiente para reutilizarlo en Fase 4 (cobranzas).
- Outbox real en Fase 3: `outbox_events` (§5.4) + worker simple procesa publicación (PDF + email via `MAIL_DRIVER`).
- Handlers HTTP delgados en `apps/api/transport/http/expensas.go`, `gastos.go`, `cuenta_corriente.go`.

## H3.1 — Conceptos y gastos con comprobantes

### Datos (migración 00013)
- `conceptos_expensa(id, tenant_id, consorcio_id NULL, nombre, categoria, regla, created_at, updated_at)`
  - `regla` enum: `coeficiente` (única soportada en esta fase).
  - `UNIQUE(tenant_id, consorcio_id NULLS DISTINCT, nombre)`.
  - `consorcio_id` NULL = concepto de tenant (usable por todos sus consorcios); no NULL = propio del consorcio.
- `gastos(id, tenant_id, consorcio_id, concepto_id FK, proveedor_id FK NULL, comprobante NULL, importe_cents BIGINT, fecha DATE, estado, documento_id FK NULL, created_at)`.
  - `estado` enum: `registrado`, `pagado`, `anulado`.
  - Comprobante opcional; documento adjunto opcional vía `documento_id`.
- RLS: todas las tablas con 3 políticas estándar (SELECT/INSERT/UPDATE) usando `app.current_tenant_id()`; GRANT a `consorcio_app`.
- Down completo vía `+goose Down`.

### API
- `GET /consorcios/{id}/conceptos` — lista conceptos (consorcio + tenant) — `expensas.read`.
- `POST /consorcios/{id}/conceptos` — crea concepto de consorcio — `expensas.create`.
- `GET /consorcios/{id}/gastos` — lista gastos con filtros `mes`, `proveedor_id` — `gastos.read`.
- `POST /consorcios/{id}/gastos` — crea gasto con `concepto_id` (resuelto en backend; no se confía en el string) — `gastos.manage`.
- Ajuste OpenAPI: `Gasto.concepto` string → `Gasto.concepto_id` UUID + `concepto_nombre` para display. `createGasto` requiere `concepto_id`.

### Invariantes
- Importe > 0.
- La fecha no puede ser futura (salvo decisión explícita).
- Referencia a `concepto_id`/`proveedor_id` de otro tenant o consorcio → error de validación.
- Test negativo de aislamiento: consorcio B no ve gastos ni conceptos de A.

## H3.2 — Máquina de estados de liquidación

- `liquidaciones(id, tenant_id, consorcio_id, periodo YYYYMM, vencimiento_1 DATE, vencimiento_2 DATE NULL, estado, version, total_gastos_cents, total_distribuido_cents, unidades_alcanzadas, created_at, updated_at)`.
- Estados: `borrador → calculada → confirmada → publicada → cerrada`; `anulada` con motivo y asientos compensatorios cuando ya hubo cargos.
- `UNIQUE(tenant_id, consorcio_id, periodo)` — no puede existir otra activa para el mismo período; `anulada`/`cerrada` liberan la combinación.
- `version` para optimistic concurrency (`If-Match`).

### API — `/liquidaciones`
- `POST /consorcios/{id}/liquidaciones` (periodo + vencimiento_1 [+ vencimiento_2]) → 201.
- `GET /consorcios/{id}/liquidaciones?periodo=&estado=` → página.
- `GET /liquidaciones/{id}`.
- `PATCH /liquidaciones/{id}` con `If-Match` — editar vencimientos solo en `borrador`.
- `POST /liquidaciones/{id}/calcular` (con `Idempotency-Key`) → transiciona a `calculada` y expone preview reproducible.
- `POST /liquidaciones/{id}/confirmar` (con `Idempotency-Key` + `If-Match`) → `confirmada`.
- `POST /liquidaciones/{id}/publicar` (con `Idempotency-Key`) → `publicada` + outbox.
- `POST /liquidaciones/{id}/anular` (con `If-Match` + motivo) → `anulada`.

Permisos: `expensas.create` / `expensas.read` / `expensas.confirm` / `expensas.publish`.

## H3.3 — Cálculo determinista con mayor resto

- `internal/expensas/calculo.go`: función pura `Distribuir(conceptos []ConceptoInput, ufs []UFCoef) (items, unidades, err)`.
  - Entrada: por concepto el total de gastos a repartir y la regla; por UF `codigo` y `coeficiente NUMERIC(12,8)`.
  - Mayor resto sobre centavos; tie-break por código UF ascendente (ADR-0006).
  - Coeficiente 0 → UF sin reparto (advertencia).
- Property-based tests: `suma(liquidacion_unidades.total_cents) == total_distribuido_cents`, todo `>= 0`, determinista ante reorden.
- Endpoint `calcular` persiste el preview reproducible en `liquidacion*` y devuelve `total_gastos_cents`, `total_distribuido_cents`, `diferencia_cents`, `coeficiente_total`, `unidades_alcanzadas`, `advertencias`, `unidades[]`.
  - `diferencia_cents = total_gastos - total_distribuido` (siempre 0 si `total_gastos` se reparte completo).
- Invariante: `total_distribuido_cents == Σ unidades`.

## H3.4 — Confirmación transaccional: snapshot + cargos + outbox (idempotente)

- Confirmar ejecuta en una **transacción**:
  1. Verifica `borrador`/`calculada` + version (If-Match) + consorcio dentro de scope.
  2. Congela snapshot: copia gastos → `liquidacion_gastos`, items → `liquidacion_items`, unidades → `liquidacion_unidades` + `liquidacion_unidad_items`.
  3. Genera débitos: por cada UF alcanzada inserta `charges` y `account_entries` (tipo débito) en `internal/cuenta_corriente`.
  4. Estado → `confirmada`, `version+1`.
  5. Encolea `outbox_events` `liquidacion.confirmada`.
- Idempotencia: reutilizar `idempotency_keys` (scope `expensas.confirmar`); reintento devuelve el resultado previo sin re-ejecutar.
- Si ya está `confirmada` y reintento con distinta key → 409.
- Anular desde `confirmada`: crea asientos de reversa (`reversa_de_id`) sobre cada cargo; nunca DELETE.

## H3.5 — Cuenta corriente por UF

- `internal/cuenta_corriente/`: `account_entries(id, tenant_id, unidad_id, tipo debito|credito, fecha_efectiva, debito_cents, credito_cents, moneda UUID fk currency, referencia, reversa_de_id NULL)`.
  - Invariante: `debito_cents XOR credito_cents` (`CHECK`).
  - Saldo = `Σ debit - credit`; reconstruible siempre.
- Migración `00014_account_entries.sql` (no mezclar con 00013).
- `GET /unidades/{id}/cuenta-corriente` → `saldo_cents`, `saldo_a_favor_cents`, `movimientos` (cursor).
  - Permiso `finanzas.read`. Son solo créditos si hay exceso (reversa o ajuste).
- Test negativo de aislamiento por UF.

## H3.6 — PDFs (worker) + publicación por email simulado

- Publicar: en transacción estado → `publicada` + insert `outbox_events` `liquidacion.publicada`.
- Worker (goroutine configurable `WORKER_ENABLED`, por defecto true en dev): poll `outbox_events` pendientes cada ~1s; por cada evento:
  - Genera PDF simple de la liquidación (backend puro, sin deps nuevas): encabezado consorcio, período, vencimientos, desglose por concepto, total por UF.
  - Guarda el PDF como `documentos` (tipo `liquidacion`, owner = consorcio) — reutiliza `documentos.Storage`.
  - Envía email mock vía `MAIL_DRIVER` (mailpit) a los vínculos vigentes de las UFs con direcciones.
  - Marca `outbox_events` `procesado`; reintentos con backoff hasta N intentos, luego `fallido` (log).
- Si el adapter de mail no existe: crear `internal/outbox/mailer.go` con interfaz mínima `Send(recipient, subject, body)` y drivers `mailpit`/`mock`.
- Auditoría: `liquidaciones.publish` y `outbox_events.*` en `audit_events`.

## H3.7 — Wizard de liquidación en UI

- Rutas: `/app/consorcios/:id/expensas` (listado, filtros periodo/estado en query params) y `/app/liquidaciones/:id` (wizard).
- `apps/web/src/pages/expensas/Expensas.tsx`: historial + botón nueva liquidación.
- `apps/web/src/pages/expensas/LiquidacionWizard.tsx` (§8.3, 7 pasos):
  1. Período y vencimientos.
  2. Gastos incluidos.
  3. Conceptos y reglas.
  4. Cálculo por unidad.
  5. Validación: errores bloqueantes y advertencias.
  6. Confirmación explícita (diálogo con período, importe, cantidad de UFs).
  7. Publicación y progreso documental.
- Cliente generado regenerado desde OpenAPI (`npm run gen`/make).
- Página de gastos `/app/consorcios/:id/gastos`.

## Definición de hecho por historia

- Migración+rollback verificado.
- OpenAPI + cliente generado en sync.
- Autorización del backend probada (positiva + negativa por scope).
- Property-based/unit tests del cálculo determinista.
- Prueba de aislamiento por recurso nuevo.
- `go build`, `golangci-lint`, `go test -race`, `npm run lint/test` verdes.
- UI responsive, keyboard-navigable (WCAG básico).

## Fuera del alcance (esta fase)
- Intereses, mora, segundo vencimiento con recargo.
- Cobranzas/pagos (Fase 4).
- Portal consorcista (Fase 5).
- PSP/banco.
- Validez legal de documentos.