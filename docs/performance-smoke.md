# Performance Smoke

## Objetivo
- Medir de forma rápida los endpoints críticos sin agregar infraestructura.

## Comando
- `make perf-smoke BASE_URL=http://localhost:8090 REPS=5`

## Endpoints
- `/healthz`
- `/metrics`
- `/api/v1/me`
- `/api/v1/portal`

## Lectura
- `avg_ms` muestra la latencia media por endpoint.
- `status` muestra el último código HTTP observado.

## Uso recomendado
- Antes de un go-live.
- Después de cambios en auth, observabilidad o portal.
- En staging, con un usuario de demo autenticado si querés extender el smoke a endpoints protegidos.
