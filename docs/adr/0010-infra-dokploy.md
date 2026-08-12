# ADR-0010 — Infraestructura: Dokploy + Postgres + MinIO; sin Redis en el MVP

**Estado:** Aceptado
**Fecha:** 12/08/2026
**Confirmado por el owner:** sí (12/08/2026)

## Contexto

El stack original mencionaba Redis como opcional y la sección 15 dejaba abierto el proveedor de infraestructura. El owner confirmó el patrón ya usado en el resto de sus proyectos: deploy con Dokploy. El criterio 14 del MVP exige operar sin PSP, IA, Redis, Elasticsearch ni Kubernetes.

## Decisión

- **Hosting:** Dokploy como plataforma de deploy (todos los proyectos del owner). El backend Go y la SPA React se despliegan como contenedores; Docker Compose define `local` y `staging`.
- **Base de datos:** PostgreSQL 16 administrado (contenedor en Dokploy o servicio externo alcanzable). Sin Redis en el MVP: colas, outbox y locks se resuelven en PostgreSQL.
- **Storage:** MinIO en desarrollo; adaptador S3-compatible en producción ([ADR-0008](./0008-ports-and-adapters.md)).
- **Correo:** SMTP capturable en dev; proveedor SMTP real en producción (pendiente decisión de canal, §15).
- **Secretos:** variables de entorno en Dokploy / secret manager; `.env.example` sin credenciales en el repo.
- **Entornos:** `local` (Compose), `test` (BD efímera), `staging` (equivalente a prod, sin dinero real), `production`.

## Consecuencias

- Menos infraestructura que operar; el outbox sobre Postgres cubre los reintentos acotados y el dead-letter del SLO (§10.3).
- Si un caso medido exige Redis (p. ej. rate-limit distribuido o caché con TTL alto), se agrega detrás de puerto; no es dependencia inicial.
- Documentación de restauración y backup con RPO 24h / RTO 4h como compromiso del MVP.
