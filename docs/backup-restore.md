# Backup y Restore

## Objetivo
- Respaldo lógico reproducible de PostgreSQL.
- Restauración manual ensayable para cumplir el compromiso MVP de RPO 24h / RTO 4h.

## Comandos
- Backup: `make backup-db`
- Restore: `make restore-db FILE=./backups/consorcioabierto-YYYYMMDDThhmmssZ.dump`

## Requisitos
- `DATABASE_URL_ADMIN` apuntando a una base con privilegios de dump/restore.
- `pg_dump` y `pg_restore` disponibles en PATH.

## Flujo local
1. Levantar el stack: `make up`
2. Exportar `DATABASE_URL_ADMIN`
3. Generar backup: `make backup-db`
4. Verificar que se creó un `.dump` en `./backups`
5. Restaurar: `make restore-db FILE=...`

## Sugerencia operativa
- Ejecutar un backup al menos diario.
- Probar restore en staging antes de cada go-live.
- Mantener 7 copias rotadas como mínimo en el entorno MVP.
