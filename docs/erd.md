# Modelo de datos (ERD) — MVP

Convenciones:
- Todas las tablas multi-tenant llevan `tenant_id UUID NOT NULL` (RLS, [ADR-0002](./adr/0002-postgres-rls.md)), `created_at`/`updated_at`.
- PK tipo `UUID` (gen_random_uuid()) o `BIGSERIAL` solo para catálogos no expuestos (IDs nunca viajan en API; se exponen UUID).
- Importes en `BIGINT` centavos ([ADR-0003](./adr/0003-money-integer.md)).
- Los índices compuestos comienzan por `tenant_id`.
- FK de negocio = claves compuestas `(tenant_id, ...)` para impedir cruces entre tenants.

## Identity y Tenancy

```
users(id, email_normalized UNIQUE, password_hash, name, status, mfa_enabled, created_at, updated_at)

tenants(id, name, slug UNIQUE, tax_id, status, plan_code, timezone, currency, created_at, updated_at)

memberships(id, user_id FK, tenant_id FK, status, created_at)
  UNIQUE(user_id, tenant_id)
  status = active | suspended | invited

roles(id, code UNIQUE, label)          -- platform_admin no vive aquí: es global

membership_roles(id UUID primary key, membership_id FK, role_id FK)
  UNIQUE(membership_id, role_id)

role_scopes(id, membership_role_id FK, scope_type, scope_id)
  scope_type = tenant | consorcio | uf

refresh_tokens(id, user_id, membership_id, token_hash UNIQUE, family_id, expires_at, revoked_at)
sessions(id, user_id, membership_id, session_id, ...)
idempotency_keys(key, scope, response_json, created_at) UNIQUE(key, scope)
```

## Organización (consorcios)

```
consorcios(tenant_id, id, nombre, cuit, domicilio, tipo, estado, config jsonb, created_at, updated_at)

unidades(tenant_id, consorcio_id FK(tenant_id,consorcio_id), id, codigo, tipo, superficie, coeficiente NUMERIC(12,8), estado)
  UNIQUE(tenant_id, consorcio_id, codigo)
  estado = activa | inactiva
  CHECK (coeficiente >= 0)

personas(tenant_id, id, nombre, documento, email, telefono, created_at, updated_at)
  -- documento/email normalizados

unidad_personas(tenant_id, id, unidad_id FK(tenant_id,unidad_id), persona_id FK(tenant_id,persona_id),
                vinculo, porcentaje NUMERIC(5,2), valid_from date, valid_to date NULL)
  vinculo = propietario | inquilino | apoderado
  -- histórico: nunca se sobreescribe; se cierra valid_to y se crea otro registro
  -- CHECK (valid_to IS NULL OR valid_to > valid_from)

conceptos_expensa(tenant_id, consorcio_id FK NULL, id, nombre, categoria, regla_distribucion, created_at)
  -- consorcio_id NULL => concepto de tenant
```

## Expensas

```
liquidaciones(tenant_id, consorcio_id FK, id, periodo char(6) 'YYYYMM', vencimiento_1 date, vencimiento_2 date NULL,
              estado, version int, total_gastos_cents BIGINT, total_distribuido_cents BIGINT,
              regla_redondeo text, config jsonb, created_by, created_at, updated_at)
  UNIQUE(tenant_id, consorcio_id, periodo) parcial sobre liquidaciones activas (no borrador+activas)
  estado = borrador | calculada | confirmada | publicada | cerrada | anulada

gastos(tenant_id, consorcio_id FK, proveedor_id FK NULL, id, comprobante, concepto, importe_cents BIGINT,
       fecha, estado, documento_id FK NULL, created_by, created_at)

liquidacion_gastos(tenant_id, liquidacion_id FK, gasto_id FK, importe_cents BIGINT)  -- snapshot del gasto
liquidacion_items(tenant_id, liquidacion_id FK, concepto_id FK, regla_aplicada, importe_cents BIGINT)
liquidacion_unidades(tenant_id, liquidacion_id FK, unidad_id FK, codigo_uf, coeficiente NUMERIC(12,8), total_cents BIGINT)
liquidacion_unidad_items(tenant_id, liquidacion_unidades_id FK, concepto_id FK, importe_cents BIGINT)
  -- invariante: SUM(liquidacion_unidades.total_cents) = liquidaciones.total_distribuido_cents
```

## Cuenta corriente y cobranzas

```
account_entries(tenant_id, unidad_id FK, id, tipo, fecha_efectiva date, debit_cents BIGINT, credit_cents BIGINT,
                currency, referencia, reverses_of FK NULL, created_by, created_at)
  tipo = cargo | pago | reversa
  CHECK (debit_cents >= 0 AND credit_cents >= 0 AND (debit_cents > 0) <> (credit_cents > 0))
  -- saldo UF = SUM(debit_cents - credit_cents), reconstruible

charges(tenant_id, unidad_id FK, liquidacion_id FK NULL, id, concepto, due_date date, total_cents BIGINT,
        saldo_cents BIGINT, created_by, created_at)
  -- saldo_cents es caché del pendiente, derivable de payment_allocations

payments(tenant_id, unidad_id FK, id, fecha, canal, importe_cents BIGINT, referencia, estado, idem_key UNIQUE,
         motivo_rechazo, created_by, created_at)
  estado = pendiente_revision | acreditado | rechazado | revertido

payment_allocations(tenant_id, payment_id FK, charge_id FK, amount_cents BIGINT, created_by, created_at)
  -- invariantes: SUM(allocations) <= payment.importe_cents (acreditado); <= charge.saldo_cents salvo saldo a favor explícito
```

## Proveedores y documentos

```
proveedores(tenant_id, id, cuit, razon_social, contacto_nombre, contacto_email, contacto_telefono, estado, created_at)
  UNIQUE(tenant_id, cuit)
provider_accounts(tenant_id, proveedor_id FK, id, tipo CBU|CVU|ALIAS, cifrado, verificacion_estado, created_at)

documentos(tenant_id, owner_type, owner_id, id, tipo, storage_key UNIQUE, nombre, mime, tamano_bytes, sha256, estado_antivirus, created_by, created_at)
  estado_antivirus = pendiente | limpio | infectado | cuarentena

audit_events(id, tenant_id NULL, actor_id, actor_membership NULL, accion, recurso_type, recurso_id,
             request_id, ip, user_agent, diff jsonb, created_at)
outbox_events(id, tenant_id, correlation_id, event_type, payload jsonb, estado, intentos, next_attempt_at, created_at)
```

## Restricciones críticas
- No hay FK con `NULL` en PK; vínculos históricos viajan con `valid_to`.
- Todo `UPDATE` sobre dinero es de estado transicional; correcciones = reversas ([ADR-0004](./adr/0004-ledger-and-idempotency.md)).
- RLS aplica sobre todas estas tablas; pruebas de aislamiento por recurso.
