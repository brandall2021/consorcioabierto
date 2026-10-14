# Smoke de carga

Runner mínimo de k6 para validar que el API responde bajo una carga pequeña y que el flujo autenticado
básico sigue vivo.

## Ejecución

```sh
make loadtest
```

Variables opcionales:

- `BASE_URL` (por defecto `http://127.0.0.1:8080`)
- `LOADTEST_EMAIL` (por defecto `admin@tenant-a.com`)
- `LOADTEST_PASSWORD` (por defecto `Admin.123`)

## Qué hace

- `GET /api/v1/health`
- `POST /api/v1/auth/login`
- `GET /api/v1/me`
- `GET /api/v1/consorcios`
- `GET /api/v1/consorcios/:id`
- `GET /api/v1/consorcios/:id/liquidaciones`
- `GET /api/v1/consorcios/:id/morosidad`
- `GET /api/v1/portal`

## Umbrales

- `http_req_failed < 1%`
- `http_req_duration p95 < 800ms`
