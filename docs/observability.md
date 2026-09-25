# Observabilidad

## Señales
- Logs: `slog` JSON por defecto, con `request_id`, `trace_id`, `span_id`, `route`, `status` y latencia.
- Métricas Prometheus: `/metrics` expone requests, duración, fallos de login y contadores del worker.
- Trazas OTel: API y worker exportan spans a stdout por defecto; se pueden reenviar a un collector externo sin cambiar código.

## Métricas clave
- `http_requests_total{method,route,status}`
- `http_request_duration_seconds{method,route}`
- `login_failures_total`
- `outbox_processed_total`
- `outbox_failed_total`

## Consultas SLO sugeridas
- Error rate API: `sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m]))`
- Latencia p95: `histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[5m])) by (le, route))`
- Worker health: `rate(outbox_failed_total[5m])` vs `rate(outbox_processed_total[5m])`

## Paneles recomendados
- Requests por ruta y status.
- Latencia p50/p95 por ruta.
- Login failures por ventana de 5m.
- Outbox processed vs failed.
