# Entornos y configuración

Configuración por variables de entorno. El repo solo contiene `.env.example` sin credenciales.

## Entornos

| Entorno | Base de datos | Storage | Correo | PSP | Uso |
|---|---|---|---|---|---|
| `local` | Postgres 16 (Compose) | MinIO | Mailpit (captura, no envía) | mock | desarrollo |
| `test` | Postgres efímero (fixtures) | adaptador en memoria/MinIO | no-op | mock determinista | pruebas CI |
| `staging` | Postgres 16 | S3-compatible staging | SMTP staging (bandeja de pruebas) | mock deshabilitado | validación previa a prod |
| `production` | Postgres 16 (replicado) | S3 | SMTP real | por definir (feature flag) | producción |

Regla: el binario valida al arrancar que ningún driver `mock`/`mailpit` esté activo en producción.

## Variables principales

```env
# App
APP_ENV=local
APP_BASE_URL=http://localhost:8080
APP_REQUEST_TIMEOUT=30s

# Base de datos (pgx)
DATABASE_URL=postgres://consorcio:consorcio@localhost:5432/consorcioabierto?sslmode=disable

# Identidad / sesión
JWT_PRIVATE_KEY=            # EdDSA/RS256 PEM (solo env o secret manager)
REFRESH_TOKEN_TTL=720h      # 30 días
ACCESS_TOKEN_TTL=10m
LOGIN_MAX_ATTEMPTS=5
MFA_REQUIRED_ROLES=tenant_admin,consorcio_admin,tesorero

# Storage (S3-compatible)
STORAGE_DRIVER=minio        # minio | s3
S3_ENDPOINT=http://localhost:9000
S3_BUCKET=consorcio-docs
S3_ACCESS_KEY=
S3_SECRET_KEY=
S3_REGION=us-east-1
S3_SIGNED_URL_TTL=5m
MAX_UPLOAD_BYTES=15728640   # 15 MiB

# Correo
MAIL_DRIVER=mailpit         # mailpit | smtp
SMTP_HOST=
SMTP_PORT=587
SMTP_USER=
SMTP_PASS=
MAIL_FROM=

# PSP (mock hasta decisión §15)
PSP_DRIVER=mock

# Auditoría / observabilidad
OTEL_EXPORTER=console
OTEL_METRICS_PORT=9090
LOG_FORMAT=json
