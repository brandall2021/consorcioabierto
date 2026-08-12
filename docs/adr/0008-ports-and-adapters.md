# ADR-0008 — Integraciones desacopladas por puertos y adaptadores

**Estado:** Aceptado
**Fecha:** 12/08/2026
**Confirmado por el owner:** infraestructura Dokploy + Postgres + MinIO (12/08/2026)

## Contexto

El principio 10 exige integrar correo, storage, PSP e IA mediante puertos/interfaces. El MVP no tiene PSP real (solo PSP simulado), el correo en dev es capturable y el storage es S3-compatible. No se deben simular integraciones externas como operativas en producción (regla §1.2).

## Decisión

- Cada integración expone un **puerto** (interface Go en el paquete consumidor) con un **adaptador** real y uno de desarrollo/test:
  - **Storage:** puerto `documents.Storage` (Put/Get/Delete/PreSignURL) → adaptador MinIO (local/test) y S3 (producción). La storage key la genera el backend; el cliente nunca la envía.
  - **Correo:** puerto `comunicaciones.Mailer` → adaptador SMTP capturable (dev) y SMTP real (prod); en el entorno `local` los envíos se capturan sin enviar.
  - **PSP:** puerto `cobranzas.PSP` → adaptador `mock` (dev/test) explícito; producción queda detrás de feature flag y jamás se marca como operativo sin contrato real.
  - **IA/OCR:** no existe en el MVP; si llega, se añade como puerto propio.
- La configuración elige adaptador por variable de entorno (`STORAGE_DRIVER`, `MAIL_DRIVER`, `PSP_DRIVER`), con valores explícitos y prohibición de "mock" en producción (validación al arrancar).

## Consecuencias

- Entornos deterministas: MinIO + Mailpit + PSP mock reproducibles en Compose.
- El código de negocio no depende de SDKs concretos; se puede cambiar de proveedor sin tocar dominio.
- Riesgo: interfaces vacías (abstracts sin caso real) — se evita creando el puerto solo cuando hay un segundo adaptador real.

**Alternativas descartadas:** llamar SDKs directamente en el dominio (acoplamiento, no testeable), abstraer en exceso (interfaces sin casos reales).
