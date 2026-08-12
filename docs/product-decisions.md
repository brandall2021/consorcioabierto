# Decisiones de producto (owner) — sección 15

Estado de las decisiones que la especificación §15 pide confirmar. Las confirmadas el 12/08/2026 quedaron reflejadas en ADRs. Las pendientes se implementan con configuraciones explícitas y adaptadores simulados; nada legal o bancario se codifica como irreversible.

## Confirmadas (12/08/2026)

| Decisión | Valor | Referencia |
|---|---|---|
| Mismo email en varias administraciones | Sí — identidad global + membresías | [ADR-0005](./adr/0005-global-identity.md) |
| Regla de redondeo | Mayor resto, luego código UF | [ADR-0006](./adr/0006-rounding.md) |
| Política de imputación | FIFO a deuda vencida + asignación manual + saldo a favor explícito | [ADR-0007](./adr/0007-payments-fifo.md) |
| Infraestructura | Dokploy + PostgreSQL 16 + MinIO; sin Redis en el MVP | [ADR-0010](./adr/0010-infra-dokploy.md) |

## Pendientes (bloquean recién al inicio de Fase 1 para estos puntos)

| Decisión | Supuesto actual (reversible) | Bloquea |
|---|---|---|
| Nombre comercial y dominio | "ConsorcioAbierto" como nombre interno de trabajo; dominio por definir | Branding y URLs firmadas públicas |
| Modelo de precios, límites por plan, suspensión | `plan_code` en `tenants`, sin lógica de límites todavía | Enforce de límites |
| Reglas exactas de coeficientes | `NUMERIC(12,8)`, mayor resto (confirmado); falta definir intereses/recargo 2º vencimiento | Cálculo de intereses (Fase 3 ampliada) |
| Validez legal y formato de liquidaciones/recibos | Se genera PDF con datos del snapshot; validez legal a validar | Formato final de PDFs |
| Canales de notificación y costo | Email SMTP (simulado en dev); costos a cargo del tenant a definir | Envíos reales |
| Residencia de datos / proveedor hosting | Dokploy (propio); residencia a definir | Cumplimiento |
| Retención de documentos/auditoría/PII | Política de retención por definir | Jobs de purga |
| PSP/banco a evaluar; ¿toca fondos? | PSP mock; la plataforma solo iniciaría pagos (no custodia) | Fase PSP |
| Datos reales para migración | Ninguno; import vía CSV con validación | Migración asistida (Fase 6) |
| SLA de soporte y RTO | RPO 24h / RTO 4h como compromiso MVP (§10.3) | Contrato operativo |

## Reglas hasta resolver pendientes
- Adapters simulados por defecto (`STORAGE_DRIVER=minio`, `MAIL_DRIVER=mailpit`, `PSP_DRIVER=mock`).
- Producción valida al arrancar que no haya drivers simulados activos.
- No se publica marca de terceros como propia; no se tocan fondos reales.
