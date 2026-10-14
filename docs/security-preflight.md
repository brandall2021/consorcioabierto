# Security preflight

Chequeo mínimo para producción antes de desplegar.

## Ejecutar

```sh
make security-preflight
```

## Reglas

- `STORAGE_DRIVER=s3`
- `MAIL_DRIVER=smtp`
- `PSP_DRIVER != mock`
- `SCAN_DRIVER != mock`
- `OTEL_EXPORTER=otlp` o `none`
- Si `OTEL_EXPORTER=otlp`, `OTEL_EXPORTER_OTLP_ENDPOINT` es obligatorio

## Lectura

El comando imprime una línea por check y devuelve exit code `1` si alguna regla falla.
