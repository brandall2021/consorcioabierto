# Roadmap y backlog — ConsorcioAbierto

Regla maestra: **una fase y una historia pequeña por vez**. No avanzar a la siguiente sin aceptación explícita. Cada historia usa la plantilla de la especificación (§13) y el Definition of Done (§12).

## Fase 0 — Descubrimiento y contrato (en curso)

**Salida:** ADRs (docs/adr/), mapa de dominios, ERD, matriz de permisos, OpenAPI inicial y este backlog. El prototipo navegable se difiere a Fase 1 (shell web) y las integraciones se deciden con el owner (§15).

- [x] Especificación v2.0 guardada como fuente de verdad
- [x] ADRs 0001–0010
- [x] Mapa de dominios
- [x] ERD
- [x] Matriz de permisos
- [x] OpenAPI inicial (32 paths, convenciones §7.1)
- [x] Decisiones del owner documentadas
- [ ] **Gate:** revisión de este paquete por el owner antes de Fase 1

## Fase 1 — Fundación

Repositorio, Compose, CI, migraciones, identidad, tenants, membresías, RBAC, RLS, auditoría, manejo de errores y shell web.

**Demo:** tenant A y B; prueba automática demuestra aislamiento.

- H1.1 Scaffolding: módulos Go (chi, pgx, sqlc, goose), apps/web (Vite + React + Tailwind), Makefile, Compose (Postgres + MinIO + Mailpit), `.env.example`, CI (lint, test, build, contrato OpenAPI + cliente). ✅ (2026-08-12)
- H1.2 Migraciones base + RLS: `users`, `tenants`, `memberships`, roles/scopes, tablas de soporte; políticas RLS y sesión `app.tenant_id`.
- H1.3 Identidad: login Argon2id, refresh opaco rotativo, MFA (TOTP), límite de intentos.
- H1.4 Tenancy: membresías, select-tenant, permisos por membresía (caché), middleware de autorización.
- H1.5 Auditoría append-only + request ID + manejo de errores RFC 9457.
- H1.6 Shell web: login, selección de tenant, layout con navegación MVP (§8.1), rutas con feature flags.

## Fase 2 — Organización

Consorcios, UFs, personas/vínculos, importación CSV, proveedores y documentos.

**Demo:** importar 500 UFs con preview, errores por fila y resultado auditable.

- H2.1 Consorcios CRUD (tenant-scoped).
- H2.2 Unidades + personas + vínculos históricos (`unidad_personas`).
- H2.3 Importación CSV: plantilla versionada, preview, errores por fila, job, modos crear/actualizar.
- H2.4 Proveedores (CUIT validado, duplicados).
- H2.5 Documentos: upload intents, virus scan (mock), storage S3/MinIO, URLs firmadas.
- H2.6 UI de las pantallas correspondientes (responsive, WCAG).

## Fase 3 — Expensas

Gastos, conceptos, cálculo determinista, snapshots, cuenta corriente, PDF y publicación por email simulado.

**Demo:** liquidar un período; suma por UF coincide con total y el reintento no duplica cargos.

- H3.1 Conceptos y gastos con comprobantes.
- H3.2 Máquina de estados de liquidación (borrador → calculada → confirmada → publicada → cerrada/anulada).
- H3.3 Cálculo determinista con mayor resto (property-based tests) + preview.
- H3.4 Confirmación transaccional: snapshot + cargos + outbox (idempotente).
- H3.5 Cuenta corriente por UF (account_entries, reversas).
- H3.6 PDFs de liquidación/recibo (worker) + publicación por email simulado.
- H3.7 Wizard de liquidación en UI (§8.3).

## Fase 4 — Cobranzas

Registro manual/importación, detección de duplicados, asignación, saldo a favor, recibos y morosidad.

**Demo:** pago parcial, pago excedente, reversa y reconstrucción de saldo.

- H4.1 Cobranzas manuales + importación; detección de referencia duplicada.
- H4.2 Acreditación y asignación FIFO (proposal + confirmación manual).
- H4.3 Saldo a favor y reversas.
- H4.4 Recibos PDF y dashboard de morosidad.
- H4.5 UI de cobranzas y cuenta corriente.

## Fase 5 — Portal y operación

Portal consorcista, comunicaciones, reclamos, notificaciones y observabilidad.

**Demo:** el consorcista ve solo sus UFs y completa un reclamo de punta a punta.

- H5.1 Portal consorcista (`/portal`): deuda, comunicados, reclamos, recibos.
- H5.2 Comunicados y envíos (outbox + worker).
- H5.3 Reclamos (máquina de estados, mensajes, adjuntos, SLA).
- H5.4 Observabilidad completa: slog JSON, OTel, métricas Prometheus, dashboards del SLO §10.3.

## Fase 6 — Piloto y hardening

- Carga, accesibilidad (Axe 360/768/1280/1440), backup/restore con RPO 24h/RTO 4h, seguridad, migración real asistida, soporte, checklist de go-live.

## Posteriores (requieren spec y threat model propios)

PSP/banco real, pagos a proveedores, IA/OCR, laboral FATERYH, barrios, app nativa, amenities, votaciones.
