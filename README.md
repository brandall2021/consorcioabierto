# ConsorcioAbierto

SaaS multiempresa para administración de consorcios de Argentina. Un backend Go (monolito modular), una SPA React y PostgreSQL 16 con aislamiento multi-tenant por RLS.

> **Estado:** Fase 1 — Fundación. H1.1 a H1.3 completas: scaffolding, identidad con MFA y límite de intentos. Ver [`docs/roadmap.md`](./docs/roadmap.md).

## Stack

- **Backend:** Go + `chi`, PostgreSQL 16, `pgx` + `sqlc`, `goose`, OpenAPI 3.1, `slog`.
- **Frontend:** React + TypeScript estricto + Vite, TanStack Query, Tailwind CSS, cliente tipado generado desde OpenAPI.
- **Infra:** Docker Compose local (Postgres, MinIO, Mailpit), deploy con Dokploy.
- **CI:** GitHub Actions (`lint` + `test -race` + build + migración contra PG real + contrato OpenAPI/cliente).

## Especificación

La fuente principal de verdad es [`consorcioabierto-especificacion-mejorada-para-ia.md`](./consorcioabierto-especificacion-mejorada-para-ia.md). Si los borradores originales la contradicen, prevalece esta versión.

## Documentación

| Documento | Ruta |
|---|---|
| Especificación | `consorcioabierto-especificacion-mejorada-para-ia.md` |
| Decisiones de arquitectura (ADRs 0001–0010) | `docs/adr/` |
| Mapa de dominios | `docs/domain-map.md` |
| Modelo de datos (ERD) | `docs/erd.md` |
| Matriz de permisos | `docs/permission-matrix.md` |
| Contrato HTTP (OpenAPI 3.1) | `api/openapi.yaml` |
| Backlog y roadmap | `docs/roadmap.md` |
| Decisiones de producto (owner) | `docs/product-decisions.md` |
| Entornos y configuración | `docs/environments.md` |

## Estado por fase

### Fase 0 — Descubrimiento y contrato ✅

Especificación v2.0 como fuente de verdad, ADRs 0001–0010, mapa de dominios, ERD, matriz de permisos, OpenAPI inicial y backlog. Commit `007ca97`.

### Fase 1 — Fundación (en curso)

- **H1.1 Scaffolding ✅** (`19a59fa`): módulos Go (`internal/{config,logger,database,httpapi,server}` + 10 dominios), `apps/api` (migrate + http), `apps/worker`, `apps/web` (Vite + React + Tailwind + Router + TanStack Query + openapi-fetch), Makefile, Compose (Postgres + MinIO + Mailpit), `.env.example`, CI.
- **H1.2 Migraciones + RLS + auth JWT ✅** (`14c4ce2`, `bbc7d69`): esquema de identidad (`users`, `tenants`, `memberships`, `roles`, `membership_roles`, `role_scopes`, `sessions`, `refresh_tokens`, `idempotency_keys`), RLS por tenant con contexto `app.current_user_id()`/`app.current_tenant_id()`, rol `consorcio_app` no dueño. Login Argon2id, access token JWT RS256 (10 min), refresh token opaco con rotación por familia, detección de reuso (revoca toda la familia), logout que revoca refresh + sesión.
- **H1.3 MFA TOTP + límite de intentos ✅** (`98cc0bb`): login en dos pasos (`mfa_token` de 5 min, purpose `mfa`), endpoints `/auth/mfa/{setup,confirm,verify,disable}`, límite de 5 intentos fallidos (password o TOTP) por email+IP en 15 min persistido en `login_attempts` vía funciones SECURITY DEFINER.
- **H1.4 Tenancy — pendiente:** membresías, select-tenant, permisos por membresía (caché), middleware de autorización.
- **H1.5 Auditoría** y **H1.6 Shell web** — pendientes.

Ver [`docs/roadmap.md`](./docs/roadmap.md) para el resto del backlog. Regla: una fase y una historia pequeña por vez; no avanzar sin aceptación explícita.

## Arquitectura

- **Monolito modular** ([ADR-0001](./docs/adr/0001-modular-monolith.md)): dominios en `internal/`, servicios y transporte HTTP en `apps/api/transport/http`.
- **Multi-tenant por RLS** ([ADR-0002](./docs/adr/0002-postgres-rls.md)): políticas por fila sobre tablas; el backend nunca confía en `tenant_id` del cliente.
- **Identidad global + membresías** ([ADR-0005](./docs/adr/0005-global-identity.md)): email único global; una persona puede pertenecer a varias administraciones.
- **Sesión JWT corto + refresh opaco rotativo** ([ADR-0009](./docs/adr/0009-token-session.md)): access 10 min, refresh 30 días, replay revoca familia.
- **Dinero en `BIGINT` centavos** ([ADR-0003](./docs/adr/0003-money-integer.md)), **libro mayor e idempotencia** ([ADR-0004](./docs/adr/0004-ledger-and-idempotency.md)), **Dokploy** ([ADR-0010](./docs/adr/0010-infra-dokploy.md)).

### Identidad (H1.2 + H1.3)

```
POST /auth/login          → 200 {user, memberships, access_token} | {mfa_required, mfa_token}
POST /auth/mfa/setup      → secret + otpauth_url (requiere access)
POST /auth/mfa/confirm    → 204 habilita (valida código TOTP)
POST /auth/mfa/verify     → 200 {user, memberships, access_token} (segundo factor)
POST /auth/mfa/disable    → 204
POST /auth/refresh        → rota refresh + emite access (cookies HttpOnly)
POST /auth/logout         → 204 revoca refresh + sesión
GET  /me                  → usuario + membresía activa + permisos
GET  /memberships         → membresías del usuario
```

- Contraseñas: Argon2id (`m=65536,t=3,p=2`), formato PHC, comparación en tiempo constante.
- Access token: JWT RS256, claims `iss, sub, email, name, session_key, iat, exp` + contexto de membresía (`membership_id`, `tenant_id`, `roles`, `scope_type`, `scope_id`) y `purpose` para el token MFA; **no** incluye permisos.
- Refresh: 32 bytes aleatorios, hash SHA-256 en BD con `family_id`; reuso detectado revoca la familia completa.
- MFA: TOTP RFC 6238 (`github.com/pquerna/otp`), 6 dígitos, ventana de 30 s.
- Límite de intentos: 5 fallos en 15 min por email+IP; persistencia en `login_attempts` (append-only, acceso solo vía funciones SECURITY DEFINER).

### Seguridad y RLS

- El rol de la app (`consorcio_app`) no es dueño de las tablas: acceso acotado por GRANT + RLS.
- `users` tiene RLS: `users_visible` (SELECT abierto, necesario para el lookup de login) + `users_update` (UPDATE solo del propio usuario).
- `sessions`/`refresh_tokens`: INSERT/UPDATE acotados al usuario (`user_id = app.current_user_id()`); el lookup por hash del refresh usa una vista `app.v_refresh_tokens` del owner (no enumerable).
- Errores RFC 9457 (`application/problem+json`) con `code` estable (p. ej. `invalid_credentials`, `too_many_attempts`, `refresh_token_reused`).

## Base de datos

Migraciones goose en `db/migrations/`:

| Migración | Contenido |
|---|---|
| `00001_init` | `pgcrypto` + `app_meta` |
| `00002_identity` | Esquema de identidad (users, tenants, memberships, roles, sesiones, refresh, idempotency) |
| `00003_rls` | Esquema `app`, funciones de contexto, políticas RLS, rol `consorcio_app` |
| `00004_roles_seed` | Roles base |
| `00005_auth_rls` | Vista `app.v_refresh_tokens`, policies de sesión/refresh, GRANT DML |
| `00006_mfa` | `users.mfa_secret`, `login_attempts`, funciones SECURITY DEFINER, policies de users |

Queries tipadas con `sqlc` en `db/queries/*.sql` → generadas en `internal/database/gen` (paquete `db`). Regenerar con `sqlc generate`; el CI falla si quedan desactualizadas.

## Desarrollo local

Requisitos: Go 1.26.5 (`/usr/local/go`), `sqlc`, `golangci-lint`, Node 22+, Docker.

```sh
# 1. Infra local (Postgres, MinIO 9000/9001, Mailpit 8025/1025 — puertos configurables)
docker compose -f deploy/compose.yaml up -d

# 2. Clave JWT de desarrollo y migraciones
make dev-key                        # escribe deploy/keys/jwt_private_dev.pem
export JWT_PRIVATE_KEY="$(cat deploy/keys/jwt_private_dev.pem)"
make migrate-up

# 3. Variables de entorno (ver .env.example): DATABASE_URL, DATABASE_URL_ADMIN, APP_HTTP_ADDR, APP_BASE_URL
export APP_HTTP_ADDR=:8090 APP_BASE_URL=http://localhost:8090

# 4. API
go run ./apps/api

# 5. Web
cd apps/web && npm install && npm run dev
```

### Verificación

```sh
go test -race ./...            # backend (tests + race detector)
golangci-lint run ./...        # lint
go build ./...                 # build
make migrate-down 1            # rollback de una migración (probado en cada una)
cd apps/web && npm run lint && npm run test
make check-openapi             # regenera el cliente y valida el contrato (falla si difiere)
```

### E2E de identidad (local)

```sh
# login normal
curl -c /tmp/cj -X POST http://localhost:8090/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@tenant-a.com","password":"Admin.123"}'

# login con MFA: primer paso devuelve {mfa_required, mfa_token};
# el código TOTP se genera desde el secret (setup) con un autenticador o script.
```

Seed local (creado a mano en la BD local, no viene en el repo): `admin@tenant-a.com` / `Admin.123` (tenant A). Ver `docs/permission-matrix.md` §"Semilla sugerida" para los roles de demo.

## Deploy en Dokploy

ConsorcioAbierto se despliega en Dokploy como **dos aplicaciones separadas** (api + web). Ambas se buildean desde el repositorio de GitHub (`brandall2021/consorcioabierto`) usando los Dockerfiles de `deploy/`.

> El deploy se hace **por proyecto/env production**. La instancia Dokploy de este proyecto es `https://dokploy.softgroup.com.ar/` (API key por `x-api-key`).

### Componentes

| App | Dockerfile | Puerto | Ruta base |
|---|---|---|---|
| Backend (Go) | `deploy/Dockerfile.api` | `8080` | `/` |
| Frontend (React/Vite + nginx) | `deploy/Dockerfile.web` | `80` | `/` |

El web usa nginx que sirve los estáticos y **proxea `/api/`** hacia el backend usando `API_UPSTREAM` (por defecto `http://consorcioabierto-api-gf3qkl:8080`). El nginx usa `resolver` + `set`, por lo que **arranca aunque el backend aún no esté desplegado** (las llamadas a `/api/` devuelven `502` hasta que el backend exista).

### Proyecto y aplicaciones (IDs ya creados)

- Proyecto: `consorcioabierto` → `R-IFWoe_rMy-RxGw01UtT`
- Environment `production` → `67MjIowEN4mbxjYyvZkjm`
- App **api** (`consorcioabierto-api`): `IboPj4bK-rQiftkHj7RRM` — appName `consorcioabierto-api-gf3qkl`
- App **web** (`consorcioabierto-web`): `kayKhRRaniBbjZgL8aQH-` — appName `consorcioabierto-web-lpmq6x`
- Dominio público que sirve el web: `consorcioabierto.softgroup.com.ar` (app web, puerto 80, HTTPS Let's Encrypt)

> **Importante:** el upstream del nginx del web apunta al backend por su `appName` interno (`consorcioabierto-api-gf3qkl`). Si se recrea la app del backend, ese nombre cambia y hay que actualizar `API_UPSTREAM` en la env del web.

### Variables de entorno del backend (app `consorcioabierto-api`)

Obligatorias al arrancar (ver `internal/config/config.go`):

```env
APP_ENV=production
APP_HTTP_ADDR=:8080
APP_BASE_URL=https://consorcioabierto.softgroup.com.ar
DATABASE_URL=postgres://USER:PASS@HOST:5432/DB?sslmode=disable
DATABASE_URL_ADMIN=postgres://USER:PASS@HOST:5432/DB?sslmode=disable   # para migraciones
JWT_PRIVATE_KEY=# PEM RSA privada (PKCS8) — sensible
```

Opcionales (con default seguro):

```env
LOG_FORMAT=json
APP_REQUEST_TIMEOUT=30s
ACCESS_TOKEN_TTL=10m
REFRESH_TOKEN_TTL=720h
MFA_TOKEN_TTL=5m
LOGIN_MAX_ATTEMPTS=5
LOGIN_ATTEMPT_WINDOW=15m
MEMBERSHIP_CACHE_TTL=1m
STORAGE_DRIVER=minio      # en producción debe ser s3 (no mock)
MAIL_DRIVER=smtp          # en producción NO mailpit
PSP_DRIVER=mock           # en producción decidir canal real
```

> En `APP_ENV=production` el arranque **falla** si `STORAGE_DRIVER`/`MAIL_DRIVER`/`PSP_DRIVER` contienen `mock`/`mailpit` (validación de `config.Validate`). No declarar los cambios sin esos drivers configurados.

### Variables de entorno del frontend (app `consorcioabierto-web`)

Solo una:

```env
API_UPSTREAM=http://consorcioabierto-api-gf3qkl:8080   # backend interno (swarm DNS)
```

### Migraciones

Las migraciones NO corren solas en el deploy. Ejecutarlas con la imagen del api apuntando a `DATABASE_URL_ADMIN`:

```sh
docker run --rm -e DATABASE_URL_ADMIN='postgres://…' consorcioabierto-api migrate up
```

### Red / DNS

- Las dos apps deben vivir en la **misma red** de Dokploy para que el web resuelva al backend por `appName`.
- El dominio público `consorcioabierto.softgroup.com.ar` debe tener un **registro A** que apunte a la IP del server para que Let's Encrypt emita el certificado y el sitio sea accesible.

## Repositorio

- Remoto: `github.com/brandall2021/consorcioabierto` (privado).
- Commits convencionales con prefijo de historia: `feat(fase1/h1.3): ...`.
- Reglas para agentes/IA en [`AGENTS.md`](./AGENTS.md) (una historia por vez, backend como autoridad, dinero en centavos, prueba negativa de aislamiento, verificación real antes de declarar completado).
