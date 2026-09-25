# Go-live Checklist

## Antes de salir
- `make go-live-check`
- Verificar que `DATABASE_URL_ADMIN` esté disponible para migraciones y backups.
- Confirmar que `scripts/db-maintenance.sh backup` funciona en staging.
- Confirmar que `/metrics` responde y la observabilidad está visible.

## Despliegue
1. Desplegar API.
2. Desplegar web.
3. Correr migraciones si hay cambios pendientes.
4. Tomar un backup lógico antes de abrir tráfico.

## Después de salir
- Revisar salud del API y outbox.
- Revisar auditoría y logs de errores.
- Probar login, portal y un flujo de lectura de reclamos.

## Rollback
- Detener tráfico.
- Restaurar backup lógico si la migración o el despliegue introducen un problema de datos.
- Revertir la versión de API y web al último release estable.
