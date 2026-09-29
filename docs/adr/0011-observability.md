# ADR-0011 — Observabilidad: OpenTelemetry + Prometheus + slog

**Fecha:** 2026-09-29
**Estado:** Aceptado

## Contexto

El SLO inicial (§10.3) exige medir disponibilidad, latencia p95, errores 5xx, estado de jobs (outbox), login fallido y envíos, y que cada request lleve `request_id`. El repositorio tenía stubs (`internal/observability` no-op, `/metrics` devuelve `"ok"`) y los logs de handlers usaban el logger global por defecto en vez del slog configurado.

## Decisión

1. **Trazas con OpenTelemetry** (`go.opentelemetry.io/otel`): un span por request HTTP con método, ruta, `request_id` y status. Exportadores `console` (defecto), `otlp` (HTTPS/HTTP, requiere endpoint) y `none`. Sin semconv: se usan nombres de atributos literales para evitar la fragmentación de módulo de `otel/semconv`.
2. **Métricas con Prometheus** (`prometheus/client_golang`) servidas en `GET /metrics` del proceso API: `http_requests_total`, `http_request_duration_seconds`, `http_requests_in_flight`, `auth_login_failures_total`, gauges del pool pgx y de `outbox_events_by_state`. No se usa OTLP para métricas: el scrape Prometheus cubre dashboards y alertas.
3. **Logs con slog JSON**: nivel configurable (`LOG_LEVEL`), `slog.SetDefault` en API y worker para que los logs de handlers sean estructurados.
4. **Outbox global pese a RLS**: los conteos por tenant son inútiles para el dashboard, así que se agregó `app.outbox_metrics()` **SECURITY DEFINER** (owner `consorcio`) con un `SELECT estado, count(*)` global, ejecutable por `consorcio_app`.

## Consecuencias

- Dependencias nuevas: `open-telemetry` (~otlp, trace, sdk) y prometheus `client_golang`.
- `/metrics` queda expuesto en el mismo puerto HTTP del API (sin puerto propio; `OTEL_METRICS_PORT` es heredada y no se usa).
- El worker no expone HTTP: sus métricas de outbox se scrapean vía API sobre la misma base (función `SECURITY DEFINER`).
- Las métricas HTTP usan el patrón de ruta chi como etiqueta (`route`) para mantener acotada la cardinalidad; rutas sin match caen a la ruta literal.
- Artefactos operativos: `docs/observability.md`, reglas SLO en `deploy/prometheus/slo.rules.yml`, dashboard de referencia en `deploy/grafana/`.