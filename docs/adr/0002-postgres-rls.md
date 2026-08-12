# ADR-0002 — PostgreSQL 16 con RLS como segunda barrera de aislamiento

**Estado:** Aceptado
**Fecha:** 12/08/2026

## Contexto

El principio 2 de la especificación exige aislamiento multi-tenant: toda entidad de negocio tiene `tenant_id` y toda consulta lo filtra. La regla 7 del mensaje maestro exige una prueba negativa de aislamiento por recurso. Un solo defecto de consulta no debe filtrar datos entre tenants.

## Decisión

- PostgreSQL 16+, acceso con `pgx` + `sqlc`.
- Cada sesión de base establece `app.tenant_id` y `app.user_id` (vía `SET LOCAL`/session variable) resuelto en el middleware desde la membresía activa, nunca desde query/body.
- RLS habilitado por defecto (`ALTER TABLE ... ENABLE ROW LEVEL SECURITY`) en toda tabla multi-tenant, con políticas específicas sobre cada tabla basadas en `app.current_tenant_id()` y `app.current_user_id()`.
- Las FK de negocio se diseñan como claves compuestas o se validan transaccionalmente para impedir referencias cruzadas entre tenants.
- La capa de aplicación SIEMPRE filtra por tenant en SQL: RLS es refuerzo, no sustituto.
- El rol de aplicación no es el dueño de las tablas y está sujeto a RLS; el dueño o migraciones pueden bypassear RLS a menos que se fuerce.
- El contexto de tenant y usuario debe propagarse también a jobs y worker (regla de aislamiento 3).

## Consecuencias

- Doble barrera: consultas filtradas por el repositorio + RLS en la base.
- Prueba obligatoria de aislamiento por recurso nuevo (incluida la prueba sin `app.tenant_id`, que debe fallar).
- Costo: configurar RLS en cada migración y mantener las políticas al agregar tablas.
- El contexto de tenant debe propagarse también a jobs y worker (regla de aislamiento 3).

**Alternativas descartadas:** una base por tenant (costosa en MVP, operación compleja), aislamiento solo por aplicación (filtro en una capa, riesgo de fuga).
