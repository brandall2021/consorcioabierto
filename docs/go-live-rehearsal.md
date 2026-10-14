# Go-live rehearsal

Plantilla para documentar un ensayo de go-live en staging.

## Ejecutar

```sh
make go-live-rehearsal
```

Variables opcionales:

- `ENVIRONMENT` (por defecto `staging`)
- `BRANCH` (por defecto `main`)
- `BACKUP_DURATION_SECONDS`
- `MIGRATE_DURATION_SECONDS`
- `SMOKE_DURATION_SECONDS`
- `RESTORE_DURATION_SECONDS`
- `OUTCOME` (por defecto `pending`)

## Uso

Rellenar con los tiempos reales del ensayo y conservar el reporte junto al runbook de go-live.
