# Matriz de permisos — MVP

Autorización: `permiso + tenant + scope`. El permiso se resuelve en backend desde la membresía activa ([ADR-0009](./adr/0009-token-session.md)); la UI solo refleja lo que el backend permite.

Roles:
- `platform_admin`: global (fuera de membresía). Administra tenants/planes/soporte; no opera datos de negocio.
- `tenant_admin`: todos los consorcios y usuarios de su tenant.
- `consorcio_admin`: consorcios dentro de su scope.
- `tesorero`: cobranzas, conciliación y pagos autorizados (scope consorcio).
- `auditor`: solo lectura de finanzas y auditoría.
- `consorcista`: solo sus UFs (vínculo vigente en `unidad_personas`), sin acceso a finanzas internas.

## Matriz (permiso → rol)

| Permiso | tenant_admin | consorcio_admin | tesorero | auditor | consorcista |
|---|---|---|---|---|---|
| tenant.users.read | ✓ | – | – | – | – |
| tenant.users.manage | ✓ | – | – | – | – |
| consorcios.read | ✓ | ✓ | ✓ | ✓ | – (vía UF) |
| consorcios.manage | ✓ | ✓ | – | – | – |
| ufs.read | ✓ | ✓ | ✓ | ✓ | ✓ (scope uf) |
| ufs.manage | ✓ | ✓ | – | – | – |
| expensas.read | ✓ | ✓ | ✓ | ✓ | ✓ (solo liquidación publicada de su UF) |
| expensas.create | ✓ | ✓ | – | – | – |
| expensas.confirm | ✓ | ✓ | – | – | – |
| expensas.publish | ✓ | ✓ | – | – | – |
| gastos.read | ✓ | ✓ | ✓ | ✓ | – |
| gastos.manage | ✓ | ✓ | – | – | – |
| cobranzas.read | ✓ | ✓ | ✓ | ✓ | ✓ (solo su UF) |
| cobranzas.manage | ✓ | ✓ | ✓ | – | – |
| finanzas.read | ✓ | ✓ | ✓ | ✓ | – |
| finanzas.reconcile | ✓ | ✓ | ✓ | – | – |
| proveedores.read | ✓ | ✓ | ✓ | ✓ | – |
| proveedores.manage | ✓ | ✓ | – | – | – |
| documentos.read | ✓ | ✓ | ✓ | ✓ | ✓ (solo propios / liquidaciones publicadas de su UF) |
| documentos.manage | ✓ | ✓ | – | – | – |
| comunicaciones.read | ✓ | ✓ | ✓ | ✓ | ✓ (solo comunicados publicados de su UF) |
| comunicaciones.send | ✓ | ✓ | – | – | – |
| reclamos.read | ✓ | ✓ | ✓ | ✓ | ✓ (solo propios / de su UF) |
| reclamos.manage | ✓ | ✓ | ✓ | – | – |
| auditoria.read | ✓ | ✓ | – | ✓ | – |

## Reglas de scope

- `tenant_admin`: scope `tenant` sobre su membresía.
- `consorcio_admin`, `tesorero`: scope `consorcio` (role_scopes con `scope_id`). Pueden tener varios consorcios.
- `auditor`: scope `consorcio` de solo lectura.
- `consorcista`: scope `uf`; además debe existir vínculo **vigente** (`unidad_personas.valid_to IS NULL`) con la UF. Sin vínculo → 403.
- `platform_admin` no consume membresías: gestiona `tenants` y `plan_codes` por otro API (backoffice).

## Pruebas obligatorias de matriz (por permiso crítico)
1. Efectivo `✓` puede ejecutar la acción (200).
2. `–` recibe 403 (no solo ocultamiento de UI).
3. `consorcista` sin vínculo vigente recibe 403 aunque tenga scope de UF.
4. `tenant_admin` de tenant A no opera recursos del tenant B (ver §9.1 de la spec).
5. Cambio de membresía (otro tenant) no arrastra permisos ni caché.

## Semilla sugerida (seeds)
- `platform_admin` global para el backoffice.
- Por tenant de demo: `tenant_admin`, `consorcio_admin` (consorcio A), `tesorero` (consorcio A), `auditor`, y `consorcista` vinculado a una UF.
