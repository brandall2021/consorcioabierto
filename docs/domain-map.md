# Mapa de dominios

Contextos acotados (bounded contexts) de ConsorcioAbierto. Los límites definen los módulos de `internal/` del monolito modular ([ADR-0001](./adr/0001-modular-monolith.md)). El transporte es un único API HTTP (`/api/v1`) y las tablas son compartidas por módulos bajo las reglas de RLS ([ADR-0002](./adr/0002-postgres-rls.md)).

## Contextos

### 1. Identity (`internal/identity`)
- Usuarios globales, contraseñas (Argon2id), MFA, sesiones (access/refresh, [ADR-0009](./adr/0009-token-session.md)).
- **No es dueño** de tenants ni de scopes.

### 2. Tenancy (`internal/tenancy`)
- Tenants (administraciones), membresías, roles, scopes, permisos, selección de tenant activo, planes/suspensión.
- Autorización: evalúa `permiso + tenant + scope`.

### 3. Consorcios (`internal/consorcios`)
- Consorcios, unidades funcionales, personas, vínculos `unidad_personas`, importación CSV de UFs.

### 4. Expensas (`internal/expensas`)
- Conceptos, gastos, liquidaciones (máquina de estados), snapshot, distribución por coeficiente con mayor resto ([ADR-0006](./adr/0006-rounding.md)), cuenta corriente por UF, cargos (deuda), documentos generados (recibos/rendiciones).

### 5. Cobranzas (`internal/cobranzas`)
- Cobros (manual/importación), detección de duplicados, acreditación, asignación FIFO ([ADR-0007](./adr/0007-payments-fifo.md)), saldo a favor, recibos, morosidad.
- Puertos hacia PSP (mock en el MVP) y `finanzas.reconcile`.

### 6. Proveedores (`internal/proveedores`)
- Proveedores, CUIT, contactos, cuentas (CBU/CVU/alias protegidos), pagos a proveedores (post-MVP).

### 7. Documentos (`internal/documentos`)
- Upload intents, virus scan, storage S3-compatible, URLs firmadas, owner/autorización.
- Todos los documentos son privados; nunca enumerables.

### 8. Comunicaciones (`internal/comunicaciones`)
- Comunicados, envíos por email (simulado en dev), notificaciones, outbox de envíos.

### 9. Reclamos (`internal/reclamos`)
- Reclamos, conversación (mensajes/adjuntos), máquina de estados `abierto → en_progreso → resuelto → cerrado`, reapertura con historial, SLA.

### 10. Platform (`internal/platform`)
- `platform_admin`: alta de tenants, planes, soporte. Backoffice interno, fuera del API de negocio.

## Transversal
- **Outbox** (`internal/outbox`): eventos entre módulos y hacia el worker (documentos, notificaciones). Sobre Postgres ([ADR-0010](./adr/0010-infra-dokploy.md)).
- **Auditoría** (`internal/audit`): append-only de acciones de negocio.
- **Idempotencia** (`internal/idem`): clave + resultado de mutaciones financieras.

## Flujos principales y sus contextos
```
Alta e importación (3) → Liquidar (4) → Deuda/cargos (4) → Cobrar/asignar (5) → Comunicar (8) → Reclamos (9)
```
