# Security Preflight

## Objetivo
- Validar que el entorno de production no arranque con drivers simulados ni secretos faltantes.

## Comando
- `make security-check`

## Reglas
- `DATABASE_URL` y `DATABASE_URL_ADMIN` deben existir.
- `JWT_PRIVATE_KEY` debe existir.
- `STORAGE_DRIVER`, `MAIL_DRIVER`, `PSP_DRIVER` y `SCAN_DRIVER` no pueden ser `mock` ni `mailpit`.

## Uso recomendado
- Antes de desplegar a production.
- Después de cualquier cambio en variables de entorno o secrets.
