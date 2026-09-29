# Observabilidad

Concentra trazas (OpenTelemetry), métricas (Prometheus) y logs (slog JSON). Implementa el SLO inicial de la especificación §10.3 ([ADR-0011]).

## Endpoints

| Endpoint | Proceso | Uso |
|---|---|---|
| `GET /healthz` | API | Liveness del proceso HTTP |
| `GET /api/v1/health` | API | Consulta la base de datos |
| `GET /metrics` | API | Scrape Prometheus (texto plano) |

El worker (`apps/worker`) no sirve HTTP: sus métricas de outbox se scrapean desde el `/metrics` del API porque ambas comparten Postgres (la función `app.outbox_metrics()` es `SECURITY DEFINER` y ve todas las filas pese a RLS).

## Trazas OpenTelemetry

- Un span por request HTTP con `service.name=consorcioabierto-api`, `http.request.method`, `url.path`, `request.id` y `http.response.status_code`.
- Exportador por `OTEL_EXPORTER`:
  - `console` (defecto): stdout, con pretty print.
  - `otlp`: requiere `OTEL_EXPORTER_OTLP_ENDPOINT`; exporta por OTLP/HTTP.
  - `none`: desactiva trazas.
- `request_id` viaja en cada request ([§10.3]): `middleware.RequestID` lo genera o respeta `X-Request-ID`, y los logs de request lo incluyen.

## Métricas Prometheus

| Métrica | Tipo | Descripción |
|---|---|---|
| `http_requests_total{route,method,status}` | Counter | Requests por ruta/método/status |
| `http_request_duration_seconds{route,status}` | Histogram | Latencia server-side (p95 del SLO §10.3) |
| `http_requests_in_flight{route}` | Gauge | Requests en curso |
| `auth_login_failures_total` | Counter | Login fallidos (ver `observability.IncLoginFailure`) |
| `database_pool_connections{state}` | Gauge | Conexiones pgx total/idle/acquired |
| `database_pool_max_connections` | Gauge | Máximo de conexiones del pool |
| `outbox_events_by_state{estado}` | Gauge | Eventos de outbox: pendiente/procesando/procesado/fallido |

Pool y outbox se refrescan cada 15s (`RegisterDBPool`, `RegisterOutbox`).

## SLO inicial (§10.3)

| Objetivo | Definición operativa |
|---|---|
| Disponibilidad 99,5% mensual | Uptime del deploy; mantenimiento programado excluido y documentado |
| API p95 < 500 ms | `histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[5m])) by (le))` |
| Errores 5xx < 1% | `sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m]))` |
| Jobs visibles, reintentos acotados, DLQ | Outbox: `outbox_events_by_state{estado="fallido"}` e inspección directa de `outbox_events` |
| RPO 24 h / RTO 4 h (MVP) | Compromiso documentado del MVP: backup diario + restore ensayado en Fase 6 |

Reglas de grabación precargadas en `deploy/prometheus/slo.rules.yml`; dashboard de referencia en `deploy/grafana/consorcioabierto-dashboard.json`.

## Logs

- `slog` JSON (defecto) o texto con `LOG_FORMAT=text`; nivel con `LOG_LEVEL=debug|info|warn|error`.
- `slog.SetDefault` se aplica al arrancar API y worker para que los logs de handlers sean estructurados.
- Prohibido loguear secretos, tokens o PII ([AGENTS.md] regla 10).

## Notas de despliegue

- `OTEL_METRICS_PORT` quedó como variable heredada: las métricas se sirven en el `/metrics` del proceso API, no en un puerto propio.
- En `production`, `OTEL_EXPORTER` debe ser `otlp` o `none` (config.validate); sin endpoint no se fuerza, pero el SLO 99,5% exige métricas scrapeadas.