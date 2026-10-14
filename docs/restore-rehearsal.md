# Restore rehearsal

## Objetivo

Probar que un backup reciente se puede restaurar dentro del RTO objetivo.

## Pasos

1. Ejecutar `make backup-db`.
2. Copiar el archivo generado a un entorno de prueba vacío.
3. Ejecutar `make restore-db BACKUP_FILE=...`.
4. Verificar:
   - login
   - `GET /api/v1/me`
   - `GET /api/v1/consorcios`
   - una ruta de detalle de consorcio
5. Anotar tiempos y cualquier intervención manual.
