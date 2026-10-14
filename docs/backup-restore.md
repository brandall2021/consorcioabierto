# Backup y restore

Compromiso MVP: **RPO 24h / RTO 4h**.

## Backup local

```sh
make backup-db
```

Variables opcionales:

- `BACKUP_DIR` (por defecto `backups`)
- `BACKUP_TIMESTAMP` (útil para pruebas reproducibles)
- `POSTGRES_USER` (por defecto `consorcio`)
- `POSTGRES_DB` (por defecto `consorcioabierto`)

## Restore local

```sh
make restore-db BACKUP_FILE=backups/backup-YYYY-MM-DDTHH-MM-SSZ.sql
```

Variables opcionales:

- `COMPOSE_FILE` (por defecto `deploy/compose.yaml`)
- `POSTGRES_USER` (por defecto `consorcio`)
- `POSTGRES_DB` (por defecto `consorcioabierto`)
- `DRY_RUN=1` para imprimir el comando sin ejecutarlo

## Rehearsal

1. Tomar un backup nuevo.
2. Restaurarlo sobre una base vacía o descartable.
3. Validar que el login, `/me` y un listado principal levantan.
4. Registrar duración total y errores en `docs/restore-rehearsal.md`.
