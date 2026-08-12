# ConsorcioAbierto

SaaS multiempresa para administración de consorcios de Argentina. Un backend Go (monolito modular), una SPA React y PostgreSQL 16 con aislamiento multi-tenant por RLS.

> **Estado:** Fase 0 — descubrimiento y contrato. Este repositorio contiene la especificación, las decisiones (ADRs), el contrato OpenAPI y el backlog. No hay código de producto todavía.

## Especificación

La fuente principal de verdad es [`consorcioabierto-especificacion-mejorada-para-ia.md`](./consorcioabierto-especificacion-mejorada-para-ia.md). Si los borradores originales la contradicen, prevalece esta versión.

## Documentos de Fase 0

| Documento | Ruta |
|---|---|
| Especificación | `consorcioabierto-especificacion-mejorada-para-ia.md` |
| Decisiones de arquitectura (ADRs) | `docs/adr/` |
| Mapa de dominios | `docs/domain-map.md` |
| Modelo de datos (ERD) | `docs/erd.md` |
| Matriz de permisos | `docs/permission-matrix.md` |
| Contrato HTTP (OpenAPI 3.1) | `api/openapi.yaml` |
| Backlog y roadmap | `docs/roadmap.md` |
| Decisiones de producto (owner) | `docs/product-decisions.md` |
| Entornos y configuración | `docs/environments.md` |

## Stack decidido

- Backend: Go + `chi`, PostgreSQL 16, `pgx` + `sqlc`, `goose`, OpenAPI 3.1, `slog`, OTel.
- Frontend: React + TypeScript estricto + Vite, TanStack Query, React Hook Form + Zod, Tailwind CSS + Radix UI, cliente generado desde OpenAPI.
- Infra: Docker Compose local (Postgres, MinIO, SMTP capturable), deploy con Dokploy.

## Roadmap

Ver [`docs/roadmap.md`](./docs/roadmap.md). Regla: una fase y una historia pequeña por vez; no avanzar sin aceptación explícita.
