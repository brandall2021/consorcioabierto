# Provider Payments Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** allow a tesorero to initiate auditable, idempotent payments to proveedores without touching funds or introducing custodial balances.

**Architecture:** extend the existing `proveedores` and `cobranzas` boundaries instead of introducing a new top-level subsystem. Provider payouts live in the provider domain, persist tenant-scoped state in Postgres, and talk to a payment/bank adapter through a small interface so the rest of the code never depends on a concrete SDK. The first slice should be initiation + status + webhook acknowledgment, with tenant isolation and idempotency enforced end-to-end.

**Tech Stack:** Go 1.26.5, Postgres + goose + sqlc, Chi HTTP handlers, React + TypeScript + Vite, TanStack Query, OpenAPI-generated client.

**Spec:** `docs/superpowers/specs/2026-10-14-pagos-a-proveedores-design.md`

## Global Constraints

- The platform only initiates payments; it does not custody funds or maintain provider balances.
- Every payment command that creates provider side effects must require an `Idempotency-Key`.
- All monetary amounts stay in `BIGINT` centavos.
- Every query and command remains tenant-scoped.
- Production must fail fast if the configured payment provider is missing its required credentials.
- The first UI slice should be minimal and avoid a redesign of the provider area.

---

### Task 1: Provider payment persistence and domain model

**Files:**
- Create: `db/migrations/00030_provider_payments.sql`
- Create: `db/queries/provider_payments.sql`
- Modify: `internal/proveedores/doc.go`
- Create: `internal/proveedores/pagos.go`
- Create: `internal/proveedores/pagos_test.go`
- Generated: `internal/database/gen/provider_payments.sql.go`
- Generated: `internal/database/gen/models.go`

**Interfaces:**
- Consumes: existing tenant-scoped `proveedores` rows and `db.Queries` patterns from cobranzas.
- Produces: provider-payment DTOs, status transitions, and DB queries for create/list/get/update.

- [ ] **Step 1: Write the failing tests**

```go
func TestCreateProveedorPagoRejectsInvalidInput(t *testing.T) { /* ... */ }
func TestCreateProveedorPagoRequiresIdempotencyKey(t *testing.T) { /* ... */ }
func TestProveedorPagoTenantIsolation(t *testing.T) { /* ... */ }
```

- [ ] **Step 2: Run the targeted test and confirm it fails**

Run: `go test ./internal/proveedores -run 'TestCreateProveedorPago|TestProveedorPagoTenantIsolation' -v`
Expected: fail because the domain functions and queries do not exist yet.

- [ ] **Step 3: Add the migration and sqlc queries**

```sql
-- provider_payments table with tenant_id, consorcio_id, proveedor_id, amount_cents,
-- currency, idempotency_key, provider, provider_reference, status, error_message,
-- created_by, created_at, updated_at.
```

- [ ] **Step 4: Implement the domain helpers minimally**

```go
type ProviderPagoInput struct {
    ProveedorID string
    AmountCents int64
    Currency string
    Reference string
}
```

- [ ] **Step 5: Run the focused tests until they pass**

Run: `go test ./internal/proveedores -run 'TestCreateProveedorPago|TestProveedorPagoTenantIsolation' -v`
Expected: PASS.

- [ ] **Step 6: Commit the persistence/domain slice**

```bash
git add db/migrations/00030_provider_payments.sql db/queries/provider_payments.sql internal/proveedores/pagos.go internal/proveedores/pagos_test.go internal/proveedores/doc.go
git commit -m "feat(proveedores): add provider payment domain"
```

### Task 2: Payment adapter and config validation

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`
- Create: `internal/proveedores/pagos_adapter.go`
- Create: `internal/proveedores/pagos_adapter_test.go`
- Modify: `apps/api/main.go`

**Interfaces:**
- Consumes: the provider-payment domain from Task 1.
- Produces: a provider/bank adapter interface that can create payment instructions and report provider references/status.

- [ ] **Step 1: Write the failing adapter tests**

```go
func TestPaymentAdapterCreatesInstruction(t *testing.T) { /* ... */ }
func TestProductionRequiresPaymentCredentials(t *testing.T) { /* ... */ }
```

- [ ] **Step 2: Run the targeted tests and confirm they fail**

Run: `go test ./internal/proveedores ./internal/config -run 'TestPaymentAdapter|TestProductionRequiresPaymentCredentials' -v`
Expected: fail because the adapter/config requirements are not wired yet.

- [ ] **Step 3: Add the adapter interface and config gate**

```go
type PaymentProvider interface {
    CreatePayment(ctx context.Context, in ProviderPagoInput) (ProviderPagoResult, error)
}
```

- [ ] **Step 4: Wire the adapter into app startup**

```go
provider := proveedores.NewPaymentProvider(cfg.PaymentProviderDriver, cfg.BaseURL, cfg.PaymentProviderToken)
```

- [ ] **Step 5: Run the focused tests until they pass**

Run: `go test ./internal/proveedores ./internal/config -run 'TestPaymentAdapter|TestProductionRequiresPaymentCredentials' -v`
Expected: PASS.

- [ ] **Step 6: Commit the adapter/config slice**

```bash
git add internal/config/config.go internal/config/config_test.go internal/proveedores/pagos_adapter.go internal/proveedores/pagos_adapter_test.go apps/api/main.go
git commit -m "feat(proveedores): add payment provider adapter"
```

### Task 3: HTTP contract and OpenAPI wiring

**Files:**
- Modify: `api/openapi.yaml`
- Modify: `apps/api/transport/http/routes.go`
- Create: `apps/api/transport/http/proveedor_pagos.go`
- Create: `apps/api/transport/http/proveedor_pagos_test.go`
- Modify: `apps/api/transport/http/proveedores.go` (only if shared error mapping is reused)
- Generated: `apps/web/src/api/generated.d.ts`

**Interfaces:**
- Consumes: the provider-payment domain and adapter from Tasks 1-2.
- Produces: new HTTP endpoints for initiating and inspecting provider payments.

- [ ] **Step 1: Write the failing HTTP tests**

```go
func TestCreateProveedorPagoReturnsCreated(t *testing.T) { /* ... */ }
func TestCreateProveedorPagoRejectsMissingIdempotencyKey(t *testing.T) { /* ... */ }
func TestProveedorPagoWebhookRejectsBadSecret(t *testing.T) { /* ... */ }
```

- [ ] **Step 2: Run the focused tests and confirm they fail**

Run: `go test ./apps/api/transport/http -run 'TestCreateProveedorPago|TestProveedorPagoWebhook' -v`
Expected: fail because routes/handlers do not exist yet.

- [ ] **Step 3: Add OpenAPI paths and handlers**

```yaml
/consorcios/{id}/proveedores/{proveedorId}/pagos:
  post:
    operationId: createProveedorPago
```

- [ ] **Step 4: Regenerate the web client types**

Run: `cd apps/web && npm run generate:client`

- [ ] **Step 5: Run the focused HTTP tests until they pass**

Run: `go test ./apps/api/transport/http -run 'TestCreateProveedorPago|TestProveedorPagoWebhook' -v`
Expected: PASS.

- [ ] **Step 6: Commit the HTTP contract slice**

```bash
git add api/openapi.yaml apps/api/transport/http/proveedor_pagos.go apps/api/transport/http/proveedor_pagos_test.go apps/api/transport/http/routes.go apps/web/src/api/generated.d.ts
git commit -m "feat(api): add provider payments http contract"
```

### Task 4: Providers UI and status surfacing

**Files:**
- Modify: `apps/web/src/pages/proveedores/Proveedores.tsx`
- Create: `apps/web/src/pages/proveedores/ProveedorPagoModal.tsx`
- Create: `apps/web/src/pages/proveedores/ProveedorPagoModal.test.tsx`
- Modify: `apps/web/src/api/client.ts` if needed for new endpoint usage

**Interfaces:**
- Consumes: the new OpenAPI client operation from Task 3.
- Produces: a minimal provider payment action in the existing providers table and a status confirmation after creation.

- [ ] **Step 1: Write the failing UI test**

```tsx
test('opens the payment modal and submits a provider payment', async () => { /* ... */ })
```

- [ ] **Step 2: Run the UI test and confirm it fails**

Run: `cd apps/web && npm test -- ProviderPagoModal`
Expected: fail because the modal/component does not exist yet.

- [ ] **Step 3: Implement the minimal modal and table action**

```tsx
<button type="button" onClick={() => setShowPago(true)}>Pagar proveedor</button>
```

- [ ] **Step 4: Run the UI tests and a quick build**

Run: `cd apps/web && npm test && npm run build`
Expected: PASS.

- [ ] **Step 5: Commit the UI slice**

```bash
git add apps/web/src/pages/proveedores/Proveedores.tsx apps/web/src/pages/proveedores/ProveedorPagoModal.tsx apps/web/src/pages/proveedores/ProveedorPagoModal.test.tsx
git commit -m "feat(web): add provider payments ui"
```

### Task 5: Final verification and docs sync

**Files:**
- Modify: `docs/roadmap.md`
- Modify: `docs/product-decisions.md` if the implementation exposes a new reversible assumption
- Modify: `README.md` only if the env/config contract changed

**Interfaces:**
- Consumes: the completed provider-payment API/UI from Tasks 1-4.
- Produces: repo docs that reflect the new state and the exact commands needed to verify it.

- [ ] **Step 1: Run the full relevant test suite**

Run: `go test ./... && cd apps/web && npm run lint && npm run test && npm run build`

- [ ] **Step 2: Confirm docs match the shipped behavior**

Make sure the roadmap and product decisions reference the new provider-payment capability and any remaining gaps.

- [ ] **Step 3: Commit the documentation sync**

```bash
git add docs/roadmap.md docs/product-decisions.md README.md
git commit -m "docs: sync provider payments status"
```
