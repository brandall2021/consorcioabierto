# AGENTS.md — ConsorcioAbierto

Instrucciones para agentes de IA que trabajen en este repositorio.

## Fuente de verdad
- `consorcioabierto-especificacion-mejorada-para-ia.md` es la instrucción principal. Si los borradores la contradicen, prevalece esta versión.
- Decisiones de arquitectura en `docs/adr/`; decisiones de producto en `docs/product-decisions.md`.
- Contrato HTTP en `api/openapi.yaml` (genera el cliente del frontend; no duplicar DTOs a mano).

## Reglas obligatorias
1. Trabajar **una fase y una historia pequeña por vez** (ver `docs/roadmap.md`). No implementar módulos futuros.
2. Antes de modificar: inspeccionar git status, README, migraciones, OpenAPI y tests. No sobrescribir cambios ajenos.
3. Presentar un plan breve (archivos, invariantes, riesgos, pruebas) antes de codificar.
4. Si falta una decisión que afecta datos, seguridad, dinero o alcance → detenerse y pedir decisión. Detalles reversibles: elegir opción razonable y documentarla.
5. Backend es autoridad: nunca confiar en `tenant_id`, permiso, estado o importe enviado por el cliente.
6. Toda consulta de negocio acotada por tenant + scope, con **prueba negativa de aislamiento** por recurso nuevo.
7. Dinero en `BIGINT` centavos; transacciones para invariantes financieras; idempotencia para reintentos; reversas para correcciones.
8. Actualizar junto al código: migración (con rollback), OpenAPI, cliente generado y tests.
9. No declarar completado sin ejecutar verificaciones (informar comando y resultado real).
10. Sin datos ficticios fuera de fixtures/test; sin secretos ni PII en logs.

## Verificación
- Backend: `go test -race ./...`, `golangci-lint run ./...`, `go build ./...`
- Migraciones: `make migrate-up` / `make migrate-down 1`
- Frontend: `cd apps/web && npm run lint && npm run test`
- Contrato: validación de `api/openapi.yaml` y diff del cliente generado (el CI falla si quedan desactualizados)
- E2E: `cd apps/web && npm run test:e2e`

## Entornos
`local`/`test`/`staging`/`production` definidos en `docs/environments.md`. Mocks solo en dev/test; producción valida que no haya drivers simulados.
