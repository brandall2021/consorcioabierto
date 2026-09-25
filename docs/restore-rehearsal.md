# Restore Rehearsal

## Objetivo
- Ensayar un restore de extremo a extremo contra una base destino no productiva.

## Comando
- `make restore-rehearsal TARGET_DATABASE_URL=postgres://... BACKUP_NAME=restore-rehearsal.dump`

## Flujo
1. Generar un backup lógico local.
2. Copiarlo a una ubicación de rehearsal.
3. Ejecutar `pg_restore` contra la base destino.
4. Confirmar el mensaje `rehearsal ok`.

## Criterio de éxito
- El comando termina sin error.
- El backup de rehearsal existe.
- `pg_restore` recibe la URL de destino esperada.

## Uso recomendado
- Antes de un go-live.
- Después de cambios de migración o de schema.
