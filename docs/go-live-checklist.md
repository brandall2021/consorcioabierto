# Go-live checklist

Checklist mínima para la migración real asistida.

## Ejecutar

```sh
make go-live-checklist
```

## Orden recomendado

1. Confirmar branch y ventana.
2. Ejecutar `make security-preflight`.
3. Tomar `make backup-db`.
4. Ejecutar la migración real.
5. Correr `make loadtest` contra el entorno objetivo.
6. Tener listo `make restore-db BACKUP_FILE=...` como rollback.

## Nota

La checklist imprime comandos, no ejecuta cambios de producción automáticamente.
