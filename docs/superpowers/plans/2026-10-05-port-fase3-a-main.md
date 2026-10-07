# Port de Fase 3 (Expensas / Liquidaciones) desde `fase3/h3.2-h3.6` a `main`

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Portar a `main` lo único de Fase 3 (H3.2-H3.7) que vive en la rama `fase3/h3.2-h3.6` —máquina de estados de liquidaciones, cálculo por mayor resto, cuenta corriente, PDFs, mailer y la UI— **sin** portar el esquema duplicado de Fase 5 que la rama creó por su cuenta.

**Architecture:** Es un **port, no un diseño nuevo**: los módulos se copian de la rama con `git show` y luego se adaptan a los contratos que `main` ya evolucionó (worker de outbox reescrito, `comunicaciones` reescrito, gen generado a mano). El esquema **no cambia**: `main` ya tiene las migraciones 00014/00015 (liquidaciones y cuenta corriente) con contenido idéntico al de la rama, y sus tablas de outbox/comunicados/reclamos son **superset** de las de la rama.

**Tech Stack:** Go (chi) + PostgreSQL 16 con RLS + React/TS (Vite, Tailwind). Monorepo monomodular, backend es autoridad.

**Spec:**
- Fuente funcional: `consorcioabierto-especificacion-mejorada-para-ia.md` (§8.3 wizard de liquidación, estados de §7).
- Roadmap: `docs/roadmap.md` → H3.2, H3.3, H3.4, H3.5, H3.6, H3.7.
- Contrato: `api/openapi.yaml`.
- **Plan original de la rama (referencia, 12 tareas, 2835 líneas):**
  `docs/superpowers/plans/2026-09-06-h3.2-h3.6-liquidaciones-flujo-completo.md` (verlo con
  `git show fase3/h3.2-h3.6:docs/superpowers/plans/2026-09-06-h3.2-h3.6-liquidaciones-flujo-completo.md`).
  Sus tareas 1-3 (migraciones 00014/00015/00016) y 4 (queries) **ya no aplican tal cual**: ver Global Constraints.

## Global Constraints

- **Fuente de verdad:** `consorcioabierto-especificacion-mejorada-para-ia.md`. Contrato en `api/openapi.yaml`.
- **Backend es autoridad:** nunca confiar en `tenant_id`, permiso, estado ni importe que venga del cliente.
- Toda consulta de negocio acotada por tenant + scope, con **prueba negativa de aislamiento**.
- **Dinero en `BIGINT` centavos**, transacciones e idempotencia. No tocar `internal/cobranzas` (riesgo de dinero, fuera de alcance).
- **Migración + OpenAPI + cliente generado + tests se actualizan juntos** en la misma tarea.
- **`sqlc generate` está roto** (error preexistente en `db/queries/comunicaciones.sql:36`). Los métodos/modelos nuevos se agregan **a mano** en `internal/database/gen/`. No se reejecuta `make gen` sobre todo el árbol.
- **`gofmt -w` solo sobre el archivo editado**: correrlo por directorio reformatea 4 archivos ajenos en `apps/api/transport/http/`.
- **No `git push` sin confirmar el branch con el usuario.** Obligatorio antes de cada push.
- Comandos de verificación (todos deben quedar verdes al final de cada tarea que toque Go/TS):
  - `go build ./... && go vet ./... && go test -race ./...`
  - `golangci-lint run ./...` (baseline: **5 issues** todos en `internal/cobranzas/testhelpers_test.go`; no aumentar)
  - `cd apps/web && npm run lint && npm run test && npm run build`
  - `make check-openapi`
- Un commit por tarea, mensaje `feat(fase3): ...` / `fix(fase3): ...`.

### Alcance EXACTO

**PORTAR (archivos nuevos en `main`, contenido de la rama):**

| Archivo | Líneas | H |
|---|---|---|
| `db/queries/liquidaciones.sql` | 120 | H3.2-H3.4 |
| `db/queries/cuenta_corriente.sql` | 49 | H3.5 |
| `internal/database/gen/liquidaciones.sql.go` | — | — |
| `internal/database/gen/cuenta_corriente.sql.go` | — | — |
| `internal/expensas/liquidacion.go` + `_test.go` | 237 | H3.2 |
| `internal/expensas/transiciones.go` + `_test.go` | 473 | H3.4 |
| `internal/expensas/calculo.go` + `_test.go` | 159 | H3.3 |
| `internal/cuenta_corriente/cuenta_corriente.go` + `_test.go` | 237 | H3.5 |
| `internal/cuenta_corriente/morosidad.go` + `_test.go` | 59 | H3.5 |
| `internal/outbox/pdf.go` + `pdf_test.go` | 82 | H3.6 |
| `internal/outbox/mailer.go` | 55 | H3.6 |
| `apps/api/transport/http/liquidaciones.go` | 309 | H3.2-H3.4 |
| `apps/api/transport/http/cuenta_corriente.go` | 74 | H3.5 |
| `apps/api/transport/http/morosidad.go` | 45 | H3.5 |
| `apps/web/src/pages/liquidaciones/Liquidaciones.tsx` | 275 | H3.7 |
| `apps/web/src/pages/liquidaciones/LiquidacionWizard.tsx` | 301 | H3.7 |
| `apps/web/src/pages/cuenta-corriente/CuentaCorriente.tsx` | 167 | H3.7 |
| `apps/web/src/pages/morosidad/Morosidad.tsx` | 102 | H3.7 |

**DIFF (la rama modificó, `main` no; aplicar el cambio de la rama):** `apps/api/transport/http/expensas.go`, `internal/expensas/doc.go`, `sqlc.yaml`, `apps/web/vite.config.ts`, `Makefile`.

**FUERA DE ALCANCE (NO portar, aunque estén en la rama):**
- `db/migrations/00016_outbox.sql`, `db/migrations/00020_phase5.sql` — `main` ya define `outbox_events`, `comunicados`, `reclamos`, `reclamo_mensajes`, `reclamo_transiciones` en `00024`/`00025` **con más columnas** (superset). Portarlas duplicaría tablas.
- `internal/comunicaciones/comunicaciones.go` + test — versión vieja (`CreateComunicado`/`PublicarComunicado`); `main` la reemplazó por `Create`/`List`/`Publish` en `comunicados.go`.
- `internal/observability/metrics.go|tracing.go` + UI de Observabilidad — `main` ya tiene H5.4 con `00026_observability.sql`.
- `internal/cobranzas/recibos.go` + `acreditacion_test.go` + `detalle_test.go` (H4.4) — riesgo de dinero, `internal/cobranzas` queda intacto.
- Integración real de Mercado Pago (ya resuelta en `main` por otro camino).
- `scripts/*`, `docs/backup-restore.md`, `docs/go-live-checklist.md`, `docs/performance-smoke.md`, `docs/restore-rehearsal.md`, `docs/security-preflight.md`, `backups/` (área de ops).
- UI de `comunicados/Comunicados.tsx` y de observabilidad.

---

### Task 1: Worktree aislado + baseline verde

**Files:**
- Create: worktree `.worktrees/port-fase3` sobre rama nueva `port/fase3`

**Interfaces:**
- Produces: entorno donde corren todas las tareas siguientes. Rama de origen de datos: `fase3/h3.2-h3.6`.

- [ ] **Step 1: Verificar que `main` está limpio y verde**

```bash
cd /home/brandall/desarrollo/consorcioabierto
git status -sb | head -3          # debe decir "up to date" con origin/main, sin cambios
go build ./... && go vet ./... && go test -race ./... 2>&1 | grep -c '^ok'   # esperado: 16
```

- [ ] **Step 2: Crear el worktree con el skill de worktrees**

Usar `superpowers:using-git-worktrees` para crear `.worktrees/port-fase3` con rama nueva `port/fase3` desde `main`. Si el skill no está disponible, comando equivalente:

```bash
cd /home/brandall/desarrollo/consorcioabierto
git worktree add -b port/fase3 .worktrees/port-fase3 main
```

- [ ] **Step 3: Confirmar que la rama de origen es legible**

```bash
cd .worktrees/port-fase3
git show fase3/h3.2-h3.6:internal/expensas/calculo.go | head -5   # debe imprimir Go
```

- [ ] **Step 4: Commit** (no aplica: aún no hay cambios; el worktree queda listo)

---

### Task 2: Queries + `gen` + modelos + `sqlc.yaml`

**Files:**
- Create: `db/queries/liquidaciones.sql`, `db/queries/cuenta_corriente.sql`
- Create: `internal/database/gen/liquidaciones.sql.go`, `internal/database/gen/cuenta_corriente.sql.go`
- Modify: `internal/database/gen/models.go` (agregar structs), `sqlc.yaml` (registrar las 2 queries)
- Test: `go build ./...`

**Interfaces:**
- Produces: paquete `db` con los 25 métodos que consumen las tareas 3-8:
  `CreateLiquidacion`, `GetLiquidacion`, `ListLiquidaciones`, `UpdateLiquidacionCalculo`,
  `UpdateLiquidacionVencimientos`, `TransitionLiquidacion`, `AnularLiquidacion`,
  `DeleteLiquidacionGastos`, `DeleteLiquidacionItems`, `DeleteLiquidacionUnidades`,
  `InsertLiquidacionGasto`, `InsertLiquidacionItem`, `InsertLiquidacionUnidad`,
  `InsertLiquidacionUnidadItem`, `GetLiquidacionGastos`, `GetLiquidacionItems`,
  `GetLiquidacionUnidades`, `GetLiquidacionUnidadItems`,
  `InsertCharge`, `GetCharge`, `GetChargesByLiquidacion`, `UpdateChargeSaldo`,
  `InsertAccountEntry`, `GetEntriesByUnidad`, `ListMorosidadByConsorcio`.
- **Advertencia sobre el outbox:** el worker y los handlers **NO** portan métodos de outbox. `main` ya tiene la pareja canónica, más robusta que la de la rama:
  - `InsertOutboxEvent`, `ClaimPendingOutboxEvents` (con claim y reintento de `fallido`), `MarkOutboxEventProcessed`, `MarkOutboxEventFailed` — en `internal/database/gen/comunicaciones.sql.go`.
  Las tareas 3-8 que toquen outbox usan **estos** nombres (contrato del worker en Task 7), nunca `GetPendingOutboxEvents`/`MarkOutboxProcessed`/`MarkOutboxFailed` (no se portan: sin claim, sin reintento, sin `last_error`/`updated_at`).

- [ ] **Step 1: Copiar las queries desde la rama**

```bash
cd /home/brandall/desarrollo/consorcioabierto/.worktrees/port-fase3
git show fase3/h3.2-h3.6:db/queries/liquidaciones.sql > db/queries/liquidaciones.sql
git show fase3/h3.2-h3.6:db/queries/cuenta_corriente.sql > db/queries/cuenta_corriente.sql
git show fase3/h3.2-h3.6:db/queries/outbox.sql > /tmp/outbox-rama.sql
```

- [ ] **Step 2: Revisar `/tmp/outbox-rama.sql` y NO portar nada del outbox**

```bash
grep -E '^-- name:' /tmp/outbox-rama.sql
grep -E '^-- name:' db/queries/comunicaciones.sql
```

Los tres que la rama define (`GetPendingOutboxEvents`, `MarkOutboxProcessed`, `MarkOutboxFailed`) **ya tienen equivalentes en `main` con otro nombre** (`ClaimPendingOutboxEvents`, `MarkOutboxEventProcessed`, `MarkOutboxEventFailed`) y son **inferiores**: no hacen claim (doble proceso entre workers), no releen `fallido` (el backoff que escriben jamás se ejecuta) y no guardan `last_error`/`updated_at`. Regla del plan: **si `main` ya tiene equivalentes, no portar**. No se agrega ningún método de outbox ni archivo `outbox.sql`/`outbox_extra.sql.go`. Verificar con un grep que los 4 métodos de `main` existen y anotar en el reporte que se descartaron los de la rama.

- [ ] **Step 3: Escribir el test/verificación que debe fallar primero**

```bash
cd /home/brandall/desarrollo/consorcioabierto/.worktrees/port-fase3
go build ./... 2>&1 | head -20
```

Esperado: **FAIL** con `undefined: db.CreateLiquidacion` (los métodos aún no existen). Guardar la salida: es la lista a completar.

- [ ] **Step 4: Agregar los métodos generados a mano**

```bash
git show fase3/h3.2-h3.6:internal/database/gen/liquidaciones.sql.go > internal/database/gen/liquidaciones.sql.go
git show fase3/h3.2-h3.6:internal/database/gen/cuenta_corriente.sql.go > internal/database/gen/cuenta_corriente.sql.go
```

**No se crea `outbox_extra.sql.go`.** Si este archivo quedó de una corrida anterior, borrarlo (Step 2: el outbox se mantiene 100% con los métodos ya existentes de `main`).

- [ ] **Step 5: Agregar los structs a `models.go`**

```bash
grep -n 'type \(Liquidacion\|LiquidacionItem\|LiquidacionUnidad\|LiquidacionUnidadItem\|LiquidacionGasto\|Charge\|AccountEntry\|OutboxEvent\) struct' internal/database/gen/models.go
```

Los que falten se copian de la rama:

```bash
git show fase3/h3.2-h3.6:internal/database/gen/models.go | grep -A15 'type Liquidacion struct'
```

(Repetir por struct: `Liquidacion`, `LiquidacionGasto`, `LiquidacionItem`, `LiquidacionUnidad`, `LiquidacionUnidadItem`, `Charge`, `AccountEntry`. **`OutboxEvent` ya existe en `main`.**)

- [ ] **Step 6: Registrar las queries en `sqlc.yaml`**

Diferir la rama respecto de `main` y aplicar solo lo que sume:

```bash
git diff $(git merge-base main fase3/h3.2-h3.6) fase3/h3.2-h3.6 -- sqlc.yaml
```

- [ ] **Step 7: Verificar compilación**

```bash
go build ./... && go vet ./...
```

Esperado: **PASS** (o errores residuales en archivos de tareas posteriores — anotarlos).

- [ ] **Step 8: Commit**

```bash
git add db/queries/liquidaciones.sql db/queries/cuenta_corriente.sql \
        internal/database/gen/liquidaciones.sql.go internal/database/gen/cuenta_corriente.sql.go sqlc.yaml
git commit -m "feat(fase3): queries y gen de liquidaciones y cuenta corriente"
```

---

### Task 3: Dominio de liquidaciones — máquina de estados (H3.2, H3.4)

**Files:**
- Create: `internal/expensas/liquidacion.go`, `internal/expensas/transiciones.go`
- Test: `internal/expensas/liquidacion_test.go`, `internal/expensas/transiciones_test.go`
- Modify: `internal/expensas/doc.go` (agregar líneas de dominio: `liquidaciones`, `transiciones`)

**Interfaces:**
- Consumes: `db.Queries` de la Task 2.
- Produces (firmas EXACTAS de la rama, consumidas por las Tasks 7-8):

```go
// liquidacion.go
type LiquidacionDTO struct{ ... }
type LiquidacionFilter struct{ ... }
func ValidateVencimientos(v1 string, v2 *string) error
func CreateLiquidacion(ctx context.Context, q *db.Queries, consorcioID, periodo, v1 string, v2 *string) (LiquidacionDTO, error)
func GetLiquidacion(ctx context.Context, q *db.Queries, consorcioID, id string) (LiquidacionDTO, error)
func ListLiquidaciones(ctx context.Context, q *db.Queries, consorcioID string, f LiquidacionFilter) ([]LiquidacionDTO, error)
func UpdateVencimientos(ctx context.Context, q *db.Queries, consorcioID, id, v1 string, v2 *string, expectedVersion int) error
func TransitionEstado(ctx context.Context, q *db.Queries, consorcioID, id, current, next string, expectedVersion int) error
// transiciones.go
func CalcularLiquidacion(ctx context.Context, q *db.Queries, consorcioID, liquidacionID string, expectedVersion int, idemKey string) (LiquidacionDTO, error)
func ConfirmarLiquidacion(ctx context.Context, q *db.Queries, consorcioID, liquidacionID string, expectedVersion int, idemKey string) (LiquidacionDTO, error)
func PublicarLiquidacion(ctx context.Context, q *db.Queries, consorcioID, liquidacionID string, expectedVersion int, idemKey string) (LiquidacionDTO, error)
func AnularLiquidacion(ctx context.Context, q *db.Queries, consorcioID, liquidacionID string, expectedVersion int, motivo string) (LiquidacionDTO, error)
```

- [ ] **Step 1: Copiar los archivos de dominio desde la rama**

```bash
cd /home/brandall/desarrollo/consorcioabierto/.worktrees/port-fase3
for f in liquidacion.go liquidacion_test.go transiciones.go transiciones_test.go; do
  git show fase3/h3.2-h3.6:internal/expensas/$f > internal/expensas/$f
done
```

- [ ] **Step 2: Ejecutar SOLO estos tests y ver qué falla**

```bash
go test ./internal/expensas/ -run 'Transitions|Validate|Agrupar|ToUnidad|Periodo|Vencimiento' -count=1 2>&1 | tail -30
```

Esperado: errores de compilación si la rama usa `db`/`outbox` con firmas distintas a las de `main`. Cada error se resuelve **adaptando la llamada al contrato de `main`**, nunca reescribiendo la lógica.

- [ ] **Step 3: Punto de control obligatorio — `outbox.EventPublished`**

`transiciones.go` emite el evento con `outbox.EventPublished` (rama: `internal/outbox/outbox.go:25` = `"liquidacion.publicada"`). Verificar en `main`:

```bash
grep -rn 'EventPublished\|EventComunicadoPublished' internal/outbox/*.go || echo "NO EXISTEN — agregar en esta tarea"
```

**Si no existen, agregarlas ahora** (son dos líneas, no esperan a la Task 7):

```go
// internal/outbox/outbox.go — bloque de constantes
const (
	EventPublished           = "liquidacion.publicada"
	EventComunicadoPublished = "comunicado.publicado"
)
```

Y si `main` tiene el literal `case "comunicado.publicado":` en `process`, cambiarlo por `case EventComunicadoPublished:` en el mismo paso. El **cuerpo** de `processPublicacion` sí espera a la Task 7.

- [ ] **Step 4: Ejecutar el paquete completo**

```bash
go test -race ./internal/expensas/ -count=1
```

Esperado: **PASS**. Si un test falla por datos/aislamiento, corregir la implementación portada (no el test) salvo que el test contradiga la spec.

- [ ] **Step 5: Commit**

```bash
git add internal/expensas/
git commit -m "feat(fase3): dominio de liquidaciones — máquina de estados y transiciones"
```

---

### Task 4: Cálculo determinista por mayor resto (H3.3)

**Files:**
- Create: `internal/expensas/calculo.go`, `internal/expensas/calculo_test.go`

**Interfaces:**
- Consumes: tipos de `internal/expensas` de la Task 3.
- Produces (firmas EXACTAS de la rama):

```go
type ConceptoCalculo struct{ ... }
type UFCoef struct{ ... }
type ItemDistribucion struct{ ... }
type UnidadItemDistribucion struct{ ... }
type UnidadDistribucion struct{ ... }
type DistribucionResult struct{ ... }
func Distribuir(conceptos []ConceptoCalculo, ufs []UFCoef) (DistribucionResult, error)
```

- [ ] **Step 1: Copiar desde la rama**

```bash
git show fase3/h3.2-h3.6:internal/expensas/calculo.go > internal/expensas/calculo.go
git show fase3/h3.2-h3.6:internal/expensas/calculo_test.go > internal/expensas/calculo_test.go
```

- [ ] **Step 2: Correr los tests de cálculo**

```bash
go test -race ./internal/expensas/ -run 'Distribuir|Distribucion|Calculo|MayorResto' -v -count=1 2>&1 | tail -40
```

Esperado: **PASS** (son tests de lógica pura, sin BD). Si fallan por propiedad (proporción + tie-break por código), corregir la implementación.

- [ ] **Step 3: Verificar invariante de dinero — la suma por UF coincide con el total**

Los tests del paquete deben incluir esa propiedad (es el criterio de demo del roadmap: *"suma por UF coincide con total"*). Si no está, agregarlo al archivo de test:

```go
func TestDistribucionSumaIgualTotal(t *testing.T) { /* escrito con los tipos reales de calculo.go */ }
```

- [ ] **Step 4: Commit**

```bash
git add internal/expensas/calculo.go internal/expensas/calculo_test.go
git commit -m "feat(fase3): cálculo determinista por mayor resto con tie-break por código"
```

---

### Task 5: Cuenta corriente y morosidad (H3.5)

**Files:**
- Create: `internal/cuenta_corriente/cuenta_corriente.go`, `morosidad.go`, ambos tests

**Interfaces:**
- Consumes: `db.Queries` de la Task 2 (`InsertCharge`, `GetCharge`, `GetChargesByLiquidacion`, `UpdateChargeSaldo`, `InsertAccountEntry`, `GetEntriesByUnidad`, `ListMorosidadByConsorcio`).
- Produces (firmas EXACTAS de la rama, consumidas por la Task 7):

```go
// cuenta_corriente.go
type ChargeDTO struct{ ... }
type AccountEntryDTO struct{ ... }
type UnidadCobro struct{ ... }
func CreateCargos(ctx context.Context, q *db.Queries, unidades []UnidadCobro, currency string) ([]ChargeDTO, []AccountEntryDTO, error)
func RevertirCargo(ctx context.Context, q *db.Queries, chargeID, unidadID, currency string) (AccountEntryDTO, error)
func ListEntriesByUnidad(ctx context.Context, q *db.Queries, unidadID string) ([]AccountEntryDTO, error)
// morosidad.go
type MorosidadDTO struct{ ... }
func ListMorosidadByConsorcio(ctx context.Context, q *db.Queries, consorcioID string) ([]MorosidadDTO, error)
// errores exportados que usan los handlers: cc.ErrCargoInvalido
```

- [ ] **Step 1: Copiar el paquete completo**

```bash
mkdir -p internal/cuenta_corriente
for f in cuenta_corriente.go cuenta_corriente_test.go morosidad.go morosidad_test.go; do
  git show fase3/h3.2-h3.6:internal/cuenta_corriente/$f > internal/cuenta_corriente/$f
done
```

- [ ] **Step 2: Correr el paquete**

```bash
go test -race ./internal/cuenta_corriente/ -count=1 2>&1 | tail -20
```

- [ ] **Step 3: Revisar el aislamiento por tenant (prueba negativa obligatoria)**

El plan exige *"toda consulta de negocio acotada por tenant + scope, con prueba negativa"*. Verificar que los tests incluyen un caso de `tenant B` no ve cargos del `tenant A`; si no, agregarlo al test.

- [ ] **Step 4: Commit**

```bash
git add internal/cuenta_corriente/
git commit -m "feat(fase3): cuenta corriente por UF y morosidad con aislamiento por tenant"
```

---

### Task 6: PDF mínimo + mailer (H3.6)

**Files:**
- Create: `internal/outbox/pdf.go`, `internal/outbox/pdf_test.go`, `internal/outbox/mailer.go`
- Modify: `internal/outbox/outbox.go` (declaración de `MailDriver` y `PDFGen`)

**Interfaces:**
- Consumes: nada de fuera del paquete (PDF sin dependencias externas).
- Produces: `(*SimplePDFGenerator).Generate(p LiquidacionPayload) ([]byte, error)`,
  `escapePDF`, `(*MailpitDriver).Send(to, subject, body string) error`, `(*MockDriver).Send(...)`.

**REGLA DE FUSIÓN (hallazgo verificado — la que más errores cuesta):**
`main` **ya declara los tipos** en `internal/outbox/outbox.go:18-24`, pero **vacíos, sin métodos**:

```go
type MailDriver interface{}                 // ← interface vacía, sin Send
type MockDriver struct{ Log *slog.Logger }  // ← sin Send
type MailpitDriver struct{ BaseURL string } // ← sin Send
type SimplePDFGenerator struct{}            // ← sin Generate
```

y ya los instancia: `apps/api/main.go:95` y `apps/worker/main.go:39` (`PDFGen: &outbox.SimplePDFGenerator{}`).
La rama declara **los mismos tipos otra vez** en `pdf.go`/`mailer.go`. **Nunca duplicar declaraciones** — portar solo los métodos.

- [ ] **Step 1: Copiar `pdf.go` y `pdf_test.go` y quitar la declaración duplicada**

```bash
git show fase3/h3.2-h3.6:internal/outbox/pdf.go > internal/outbox/pdf.go
git show fase3/h3.2-h3.6:internal/outbox/pdf_test.go > internal/outbox/pdf_test.go
grep -n 'type SimplePDFGenerator' internal/outbox/pdf.go internal/outbox/outbox.go
```

Si `pdf.go` trae `type SimplePDFGenerator struct{}`, **borrar esa línea de `pdf.go`** (queda la de `outbox.go`). Dejar `LiquidacionPayload`, `Generate` y `escapePDF`.

- [ ] **Step 2: Copiar `mailer.go` y quitar las declaraciones duplicadas**

```bash
git show fase3/h3.2-h3.6:internal/outbox/mailer.go > internal/outbox/mailer.go
grep -n 'type MailDriver\|type MockDriver\|type MailpitDriver' internal/outbox/mailer.go internal/outbox/outbox.go
```

Borrar de `mailer.go` las tres líneas `type ... struct/interface` que ya están en `outbox.go`. Deben quedar **solo** `func (m *MailpitDriver) Send(...)` y `func (m *MockDriver) Send(...)`, más lo auxiliar que usen.

- [ ] **Step 3: Dar método a `MailDriver` (cambio atómico, paso obligatorio)**

Sin este paso los `Send` portados no encajan en la interface y los `main.go` dejan de compilar. Editar `internal/outbox/outbox.go`:

```go
// antes
type MailDriver interface{}
// después
type MailDriver interface {
	Send(to, subject, body string) error
}
```

- [ ] **Step 4: Tipar `PDFGen` en el struct `Worker`**

```go
// internal/outbox/outbox.go — struct Worker
PDFGen PDFGenerator   // era `any`; declarar `type PDFGenerator interface{ Generate(...) }` con la firma real de pdf.go
```

Usar exactamente la firma de `Generate` que trajo `pdf.go` (incluido el tipo de payload).

- [ ] **Step 5: Verificar que todo el módulo compila junto**

```bash
go build ./... 2>&1 | head -20
go test -race ./internal/outbox/ -count=1 2>&1 | tail -20
```

Esperado: **PASS**. Si falla por un tipo que `main` no tiene (`LiquidacionPayload` importado desde otro lado, o `Mail` usado con tipo distinto), adaptar la declaración de `main`, **nunca** duplicar el tipo.

- [ ] **Step 6: Commit**

```bash
git add internal/outbox/
git commit -m "feat(fase3): PDF de liquidación y envío de mail del worker"
```

---

### Task 7: Worker — evento `liquidacion.publicada` (H3.4 + H3.6)

**Files:**
- Modify: `internal/outbox/outbox.go`, `apps/worker/main.go`

**Interfaces:**
- Consumes: `expensas`, `outbox.PDFGenerator`, `db.Queries` de tareas anteriores.
- Produces: el caso `case EventPublished: w.processPublicacion(...)`.

- [ ] **Step 1: Ver el estado actual de `main`**

```bash
grep -n 'case "' internal/outbox/outbox.go       # solo: case "comunicado.publicado"
grep -n 'func (w \*Worker)' internal/outbox/outbox.go
```

`main` tiene `Run`, `drain`, `process`, `markFailed` (122 líneas, con transacción `tx`). La rama tiene además `processBatch`, `processEvent`, **`processPublicacion`**, `processComunicado` (255 líneas).

- [ ] **Step 2: Agregar las constantes de evento**

```go
// internal/outbox/outbox.go — arriba, junto a las demás constantes
const (
	EventPublished           = "liquidacion.publicada"
	EventComunicadoPublished = "comunicado.publicado"
)
```

Además, cambiar el `case "comunicado.publicado"` literal por `case EventComunicadoPublished`.

- [ ] **Step 3: Portar `processPublicacion` desde la rama**

```bash
git show fase3/h3.2-h3.6:internal/outbox/outbox.go | sed -n '153,181p'
```

Copiar esa función a `internal/outbox/outbox.go` **adaptando la firma** a como `main` construye `q` y `tx` dentro de `process` (leer `process` de `main` primero: recibe `ctx`, `event`, arma `q`, ejecuta el case, confirma `tx`).

- [ ] **Step 4: Agregar el case en `process`**

```go
case EventPublished:
	if err := w.processPublicacion(ctx, q, ev); err != nil {
		return err
	}
```

- [ ] **Step 5: Wirear `PDFGen` y `Mail` en `apps/worker/main.go`**

```bash
grep -n 'PDFGen\|Mail:\|outbox.Worker{' apps/worker/main.go
```

Debe quedar `PDFGen: &outbox.SimplePDFGenerator{}` y el `MailDriver` configurado según `cfg.MailDriver` (el patrón ya está en `main`: `Mail: ...`). El diff de la rama es la referencia exacta:

```bash
git diff $(git merge-base main fase3/h3.2-h3.6) fase3/h3.2-h3.6 -- apps/worker/main.go
```

- [ ] **Step 6: Correr todos los tests**

```bash
go build ./... && go vet ./... && go test -race ./internal/outbox/ ./internal/expensas/ ./internal/cuenta_corriente/ -count=1
```

Esperado: **PASS**.

- [ ] **Step 7: Commit**

```bash
git add internal/outbox/outbox.go apps/worker/main.go
git commit -m "feat(fase3): worker procesa liquidacion.publicada con PDF y mail"
```

---

### Task 8: Handlers HTTP + rutas + wiring

**Files:**
- Create: `apps/api/transport/http/liquidaciones.go`, `cuenta_corriente.go`, `morosidad.go`
- Modify: `apps/api/transport/http/routes.go`, `apps/api/transport/http/expensas.go`, `internal/server/server.go`, `apps/api/main.go`

**Interfaces:**
- Consumes: dominios de las Tasks 3-5 (`expensas.*`, `cc.*`), `httpapi`, `audit`.
- Produces: handlers `ListLiquidaciones`, `GetLiquidacion`, `CreateLiquidacion`, `PatchLiquidacion`,
  `CalcularLiquidacionHandler`, `ConfirmarLiquidacionHandler`, `PublicarLiquidacionHandler`,
  `AnularLiquidacionHandler`, `GetCuentaCorriente`, `GetMorosidad`.

- [ ] **Step 1: Copiar los tres handlers nuevos**

```bash
for f in liquidaciones.go cuenta_corriente.go morosidad.go; do
  git show fase3/h3.2-h3.6:apps/api/transport/http/$f > apps/api/transport/http/$f
done
```

- [ ] **Step 2: Aplicar el diff de `expensas.go` (la rama lo extendió)**

```bash
git diff $(git merge-base main fase3/h3.2-h3.6) fase3/h3.2-h3.6 -- apps/api/transport/http/expensas.go
```

Aplicar solo lo que sume respecto de `main` (no pisar nada que `main` ya tenga).

- [ ] **Step 3: Verificar compilación y ver la lista de errores**

```bash
go build ./... 2>&1 | head -30
```

- [ ] **Step 4: Agregar el bloque de rutas de Fase 3 en `routes.go`**

Insertar después del bloque de Expensas existente (línea ~155 de `main`, el que cierra con `gr.Group(...)`), **adaptado al formato actual**:

```go
	// Liquidaciones: lectura con expensas.read, creación/cálculo con expensas.create,
	// confirmar/anular con expensas.confirm y publicar con expensas.publish.
	r.Group(func(lr chi.Router) {
		lr.Use(RequirePermission(h.Manager, "expensas.read"))
		lr.Get("/consorcios/{id}/liquidaciones", h.ListLiquidaciones)
		lr.Get("/consorcios/{consorcioId}/liquidaciones/{liquidacionId}", h.GetLiquidacion)
		lr.Group(func(mgmt chi.Router) {
			mgmt.Use(RequirePermission(h.Manager, "expensas.create"))
			mgmt.Post("/consorcios/{id}/liquidaciones", h.CreateLiquidacion)
			mgmt.Patch("/consorcios/{consorcioId}/liquidaciones/{liquidacionId}", h.PatchLiquidacion)
			mgmt.Post("/consorcios/{consorcioId}/liquidaciones/{liquidacionId}/calcular", h.CalcularLiquidacionHandler)
			mgmt.Group(func(conf chi.Router) {
				conf.Use(RequirePermission(h.Manager, "expensas.confirm"))
				conf.Post("/consorcios/{consorcioId}/liquidaciones/{liquidacionId}/confirmar", h.ConfirmarLiquidacionHandler)
				conf.Post("/consorcios/{consorcioId}/liquidaciones/{liquidacionId}/anular", h.AnularLiquidacionHandler)
			})
			mgmt.Group(func(pub chi.Router) {
				pub.Use(RequirePermission(h.Manager, "expensas.publish"))
				pub.Post("/consorcios/{consorcioId}/liquidaciones/{liquidacionId}/publicar", h.PublicarLiquidacionHandler)
			})
		})
	})
```

Y, dentro del grupo de cobranzas existente (junto a `cr.Get("/consorcios/{id}/morosidad", ...)` — verificar que **no** exista ya):

```go
		cr.Get("/consorcios/{id}/unidades/{unidadId}/cuenta-corriente", h.GetCuentaCorriente)
		cr.Get("/consorcios/{id}/morosidad", h.GetMorosidad)
```

> **Permiso nuevo:** `expensas.confirm` y `expensas.publish`. Verificar que existan en el catálogo de permisos de `main` (`grep -rn 'expensas.publish\|expensas.confirm' internal/ apps/api/transport/http/routes.go`). Si no existen, agregarlos donde `main` declara los permisos, con su prueba.

- [ ] **Step 5: Wiring del server**

```bash
grep -n 'NewAuthHandlers\|type Server struct' internal/server/server.go apps/api/main.go | head
git diff $(git merge-base main fase3/h3.2-h3.6) fase3/h3.2-h3.6 -- internal/server/server.go apps/api/main.go | head -60
```

Aplicar solo lo necesario para que los handlers nuevos existan en el server (nada de observability/tracing — fuera de alcance).

- [ ] **Step 6: Verificar que la ruta responde (aunque sea 401 sin token)**

```bash
go build ./... && go vet ./...
cd apps/web && npm run lint 2>&1 | tail -3 && cd ../..
```

- [ ] **Step 7: Commit**

```bash
git add apps/api/transport/http/ internal/server/ apps/api/main.go
git commit -m "feat(fase3): handlers HTTP y rutas de liquidaciones, cuenta corriente y morosidad"
```

---

### Task 9: OpenAPI + cliente generado

**Files:**
- Modify: `api/openapi.yaml`
- Modify: `apps/web/src/api/generated.d.ts`

**Interfaces:**
- Consumes: firmas de los handlers de la Task 8.
- Produces: schemas `Liquidacion`, `LiquidacionInput`, rutas de liquidaciones/cuenta-corriente/morosidad.

- [ ] **Step 1: Extraer los paths de la rama y verificar cuáles faltan**

```bash
OP=api/openapi.yaml
git show fase3/h3.2-h3.6:api/openapi.yaml | grep -E '^  /' > /tmp/paths-rama.txt
grep -E '^  /' $OP > /tmp/paths-main.txt
comm -23 <(sort /tmp/paths-rama.txt) <(sort /tmp/paths-main.txt)
```

Los que falten (esperado: los 7 de liquidaciones + `/consorcios/{id}/unidades/{unidadId}/cuenta-corriente` + `/consorcios/{id}/morosidad`) se copian con su bloque completo desde la rama:

```bash
git show fase3/h3.2-h3.6:api/openapi.yaml > /tmp/openapi-rama.yaml
```

Copiar también los `schemas` referenciados (`Liquidacion`, `LiquidacionInput`) y sus `$ref`.

- [ ] **Step 2: Verificar el contrato**

```bash
make check-openapi
```

Esperado: **PASS**. Si falla, corregir `openapi.yaml` (nunca desactivar la chequeada).

- [ ] **Step 3: Regenerar el cliente**

```bash
grep -n 'gen-client\|openapi-typescript\|generate' Makefile | head -5
```

Usar el comando del `Makefile` (no inventar uno). Luego:

```bash
git diff --stat apps/web/src/api/generated.d.ts
```

- [ ] **Step 4: Commit**

```bash
git add api/openapi.yaml apps/web/src/api/generated.d.ts
git commit -m "feat(fase3): contrato OpenAPI de liquidaciones y cliente regenerado"
```

---

### Task 10: UI (H3.7)

**Files:**
- Create: `apps/web/src/pages/liquidaciones/Liquidaciones.tsx`, `LiquidacionWizard.tsx`
- Create: `apps/web/src/pages/cuenta-corriente/CuentaCorriente.tsx`
- Create: `apps/web/src/pages/morosidad/Morosidad.tsx`
- Modify: `apps/web/src/router.tsx`, `apps/web/src/components/layout/AppLayout.tsx`
- Create: tests de las pantallas nuevas
- Modify: `apps/web/vite.config.ts` (solo si la rama lo cambió por algo necesario)

**Interfaces:**
- Consumes: cliente generado de la Task 9 (`GetLiquidaciones`, `CreateLiquidacion`, …).
- Produces: rutas `/liquidaciones`, `/consorcios/:id/cuenta-corriente`, etc.

- [ ] **Step 1: Copiar las cuatro páginas**

```bash
for p in liquidaciones/Liquidaciones.tsx liquidaciones/LiquidacionWizard.tsx \
         cuenta-corriente/CuentaCorriente.tsx morosidad/Morosidad.tsx; do
  mkdir -p apps/web/src/pages/$(dirname $p)
  git show fase3/h3.2-h3.6:apps/web/src/pages/$p > apps/web/src/pages/$p
done
```

- [ ] **Step 2: Ver qué rutas registra la rama y añadirlas en `router.tsx` de main**

```bash
git diff $(git merge-base main fase3/h3.2-h3.6) fase3/h3.2-h3.6 -- apps/web/src/router.tsx
```

Aplicar **solo** las entradas de `liquidaciones`, `cuenta-corriente` y `morosidad` (no las de comunicados/observabilidad).

- [ ] **Step 3: Añadir la navegación en `AppLayout.tsx`**

```bash
git diff $(git merge-base main fase3/h3.2-h3.6) fase3/h3.2-h3.6 -- apps/web/src/components/layout/AppLayout.tsx
```

Aplicar solo los ítems de menú de Fase 3.

- [ ] **Step 4: Arreglar imports del cliente generado**

```bash
cd apps/web && npx tsc --noEmit -p tsconfig.app.json 2>&1 | head -20
```

Si las páginas importan funciones del cliente con nombres distintos a los de `main` (`Create`/`List`/`Publish`), **adaptar la página al cliente de `main`** (el cliente es la fuente de verdad, no la página).

- [ ] **Step 5: Escribir un test por pantalla nueva**

Mínimo: render sin crash + llamada al endpoint mockeado + estado vacío. Seguir el patrón de `apps/web/src/pages/portal/Notificaciones.test.tsx` (mock de `vi.mock` del cliente y de `useAuth`).

```bash
cd apps/web && npx vitest run src/pages/liquidaciones src/pages/cuenta-corriente src/pages/morosidad 2>&1 | tail -20
```

- [ ] **Step 6: Verificación frontend completa**

```bash
cd apps/web
npm run lint 2>&1 | tail -3      # esperado: 0 errores (2 warnings preexistentes)
npx tsc --noEmit -p tsconfig.app.json && echo "tsc OK"
npm run test 2>&1 | grep -E 'Test Files|Tests '
npm run build 2>&1 | grep 'built in'
```

- [ ] **Step 7: Commit**

```bash
git add apps/web/
git commit -m "feat(fase3): UI de liquidaciones, wizard, cuenta corriente y morosidad"
```

---

### Task 11: Verificación integral + reporte

**Files:**
- Test: todo el repo

- [ ] **Step 1: Backend completo**

```bash
cd /home/brandall/desarrollo/consorcioabierto/.worktrees/port-fase3
go build ./... && go vet ./... && echo "build+vet OK"
go test -race ./... 2>&1 | grep -c '^ok'          # esperado: >= 16
golangci-lint run ./... 2>&1 | grep -E '^[0-9]+ issues'   # esperado: 5 (mismo baseline)
```

- [ ] **Step 2: Frontend completo**

```bash
cd apps/web && npm run lint 2>&1 | tail -2 && npm run test 2>&1 | grep -E 'Test Files|Tests ' && npm run build 2>&1 | grep 'built in'
```

- [ ] **Step 3: Contrato**

```bash
cd /home/brandall/desarrollo/consorcioabierto/.worktrees/port-fase3 && make check-openapi
```

- [ ] **Step 4: Migraciones — verificar que NO se agregó ninguna y que el esquema existe**

```bash
ls db/migrations/ | wc -l                      # mismo conteo que en main
ls db/migrations/ | tail -3                    # 00027, 00028, 00029
grep -c 'liquidaciones\|charges\|account_entries' db/migrations/00014_liquidaciones.sql db/migrations/00015_cuenta_corriente.sql
```

- [ ] **Step 5: Revisar el alcance contra la tabla de "FUERA DE ALCANCE"**

```bash
git diff --name-only main...port/fase3 | sort
```

Ningún archivo de la lista "FUERA DE ALCANCE" debe aparecer (`internal/cobranzas/`, `internal/comunicaciones/comunicaciones.go`, `internal/observability/metrics.go`, `scripts/`, `backups/`, `db/migrations/00016_outbox.sql`, `db/migrations/00020_phase5.sql`, `apps/web/src/pages/observabilidad/`, `apps/web/src/pages/comunicados/`).

- [ ] **Step 6: Reportar al usuario con el resumen de commits y pedir decisión de merge/push**

**NO hacer `git push`.** El usuario debe confirmar el branch. Reportar:

```bash
git log --oneline main..port/fase3
git diff --stat main...port/fase3 | tail -3
```
