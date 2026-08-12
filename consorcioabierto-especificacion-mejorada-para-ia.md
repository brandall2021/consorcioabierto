# ConsorcioAbierto — especificación mejorada y plan de ejecución para una IA

**Versión:** 2.0
**Fecha:** 12/08/2026
**Estado:** contrato de producto e implementación
**Tecnologías decididas:** Go + PostgreSQL + React/TypeScript

> Este documento reemplaza decisiones ambiguas de las especificaciones originales. Debe usarse como instrucción principal. Los documentos originales sirven como catálogo funcional de referencia, no como orden para implementar todo simultáneamente.

---

## 1. Objetivo del producto

Construir un SaaS multiempresa para administraciones de consorcios de Argentina. Una administración (tenant) podrá operar varios consorcios; cada consorcio tendrá unidades funcionales, personas vinculadas, liquidaciones de expensas, gastos, cobranzas, proveedores, comunicaciones y reclamos.

El primer lanzamiento debe resolver bien este circuito:

1. Crear una administración y sus usuarios.
2. Crear uno o más consorcios.
3. Importar unidades funcionales y vincular propietarios/inquilinos.
4. Registrar proveedores, conceptos y gastos.
5. Preparar, calcular, confirmar y publicar una liquidación.
6. Generar la deuda y el recibo de cada unidad.
7. Registrar o importar cobros y aplicarlos a deuda.
8. Consultar saldos, morosidad, comprobantes y trazabilidad.
9. Comunicar la liquidación al consorcista.
10. Permitir reclamos y seguimiento.

El objetivo del MVP no es replicar cada característica comercial de un tercero. Es entregar un núcleo contable consistente, seguro y verificable que pueda ampliarse.

### 1.1 Usuarios

- `platform_admin`: administra tenants, planes y soporte. No pertenece a un tenant.
- `tenant_admin`: administra la empresa, usuarios y todos sus consorcios.
- `consorcio_admin`: opera los consorcios asignados.
- `tesorero`: registra cobranzas, conciliaciones y pagos autorizados.
- `auditor`: acceso de solo lectura a finanzas y auditoría.
- `consorcista`: accede únicamente a sus unidades y datos habilitados.

Los roles `guardia`, `empleado` y módulos laboral/barrios se incorporan después del MVP.

### 1.2 Fuera del MVP

- Cuenta bancaria digital propia y movimiento real de dinero.
- Pago automático a proveedores.
- Integración productiva con PSP, bancos, ARCA/AFIP o WhatsApp.
- Liquidación laboral FATERYH y Libro Ley.
- OCR/IA de facturas y asistente conversacional.
- App nativa Flutter y modo guardia offline.
- Amenities, votaciones, control de accesos y pólizas.
- Microservicios, Kubernetes, Elasticsearch o event sourcing.

Estos puntos deben implementarse detrás de interfaces y feature flags en fases posteriores. No simular que una integración externa está operativa.

---

## 2. Principios obligatorios

1. **Monolito modular:** un backend Go desplegable y una SPA React. No crear microservicios en el MVP.
2. **Aislamiento multi-tenant:** toda entidad de negocio tiene `tenant_id`; toda consulta lo filtra; PostgreSQL RLS actúa como segunda barrera.
3. **Contabilidad por libro de movimientos:** el saldo no se edita manualmente. Se deriva de débitos, créditos y aplicaciones.
4. **Dinero entero:** almacenar centavos en `BIGINT`; nunca usar `float`.
5. **Snapshots:** una liquidación confirmada congela coeficientes, conceptos, gastos y distribución.
6. **Estados explícitos:** las transiciones ocurren mediante comandos de dominio, no mediante un `PATCH estado` genérico.
7. **Idempotencia:** toda operación financiera o webhook acepta una clave idempotente.
8. **Auditoría:** registrar quién, cuándo, desde dónde y qué cambió, sin almacenar secretos.
9. **Backend como autoridad:** los permisos, cálculos y estados se validan en servidor.
10. **Integraciones desacopladas:** PSP, correo, storage e IA se acceden mediante puertos/interfaces y adaptadores.
11. **Accesibilidad y responsive:** WCAG 2.2 AA, teclado completo y vistas utilizables desde 360 px.
12. **Sin datos inventados en producción:** mocks y fixtures solo en desarrollo/test.

---

## 3. Arquitectura decidida

### 3.1 Stack

**Backend**

- Go estable actual, router `chi`.
- PostgreSQL 16+.
- `pgx` + `sqlc`; migraciones con `goose`.
- OpenAPI 3.1 como contrato versionado.
- `slog` JSON, OpenTelemetry y métricas Prometheus.
- Worker dentro del mismo binario o segundo proceso del mismo repositorio.
- Storage S3 compatible; MinIO en desarrollo.
- Redis solo cuando un caso medido lo requiera. No es dependencia inicial.

**Frontend**

- React + TypeScript estricto + Vite.
- React Router, TanStack Query y React Hook Form + Zod.
- Tailwind CSS + Radix UI; componentes propios en `components/ui`.
- Cliente generado desde OpenAPI. No duplicar manualmente DTOs.
- Vitest, Testing Library, MSW y Playwright.

### 3.2 Repositorio

```text
/
├─ apps/
│  ├─ api/                    # entrypoint HTTP Go
│  ├─ worker/                 # jobs y outbox
│  └─ web/                    # React SPA
├─ internal/
│  ├─ identity/
│  ├─ tenancy/
│  ├─ consorcios/
│  ├─ expensas/
│  ├─ cobranzas/
│  ├─ proveedores/
│  ├─ documentos/
│  ├─ comunicaciones/
│  ├─ reclamos/
│  └─ platform/
├─ db/
│  ├─ migrations/
│  ├─ queries/
│  └─ seeds/
├─ api/openapi.yaml
├─ deploy/compose.yaml
├─ docs/adr/
├─ Makefile
└─ README.md
```

Cada módulo backend se divide en `domain`, `application`, `repository` y `transport/http`. Evitar abstracciones vacías: las interfaces se ubican en el consumidor y solo existen si hay al menos un caso real.

### 3.3 Entornos

- `local`: Docker Compose, proveedor de correo capturable, MinIO y PSP simulado.
- `test`: BD efímera y adaptadores deterministas.
- `staging`: configuración equivalente a producción, sin dinero real.
- `production`: secretos desde secret manager, TLS, backups y observabilidad.

Configurar por variables de entorno. Incluir `.env.example` sin credenciales.

---

## 4. Modelo multi-tenant, identidad y permisos

### 4.1 Correcciones respecto de los borradores

- `users.email UNIQUE` global se reemplaza por identidad global + membresías, permitiendo que una persona participe en más de una administración.
- Una PK no debe contener `NULL`; separar membresía, roles y scopes.
- `platform_admin` es global; `tenant_admin` no debe confundirse con superadministración.
- No confiar en `tenant_id` enviado por query/body. Se resuelve desde la membresía activa y se cruza con el recurso.
- RLS debe implementarse en migraciones y probarse; declararla en texto no alcanza.

### 4.2 Entidades de identidad

```text
users
  id, email_normalized, password_hash, name, status, mfa_enabled, timestamps

tenants
  id, name, slug, tax_id, status, plan_code, timezone, currency, timestamps

memberships
  id, user_id, tenant_id, status, created_at
  UNIQUE(user_id, tenant_id)

roles
  id, code, label

membership_roles
  membership_id, role_id
  PRIMARY KEY(membership_id, role_id)

role_scopes
  id, membership_role_id, scope_type, scope_id
  scope_type = tenant | consorcio | uf
```

La unicidad de email debe aplicarse sobre el email normalizado global. El login devuelve las membresías disponibles; si hay más de una, el usuario elige tenant antes de recibir el access token contextual.

### 4.3 JWT y sesión

- Access token RS256/EdDSA: 10 minutos.
- Refresh token opaco: 30 días, almacenado hasheado, rotación en cada uso.
- Claims mínimos: `sub`, `membership_id`, `tenant_id`, `session_id`, `iat`, `exp`, `jti`.
- No incluir listas extensas de permisos en JWT; resolverlas y cachearlas por membresía.
- Cookies `HttpOnly`, `Secure`, `SameSite=Lax` para la web. Si se usa bearer en otro cliente, almacenarlo en mecanismo seguro del sistema operativo.
- Revocar la familia completa si se detecta replay de refresh token.

### 4.4 Permisos mínimos

```text
tenant.users.read          tenant.users.manage
consorcios.read            consorcios.manage
ufs.read                   ufs.manage
expensas.read              expensas.create
expensas.confirm           expensas.publish
gastos.read                gastos.manage
cobranzas.read             cobranzas.manage
finanzas.read              finanzas.reconcile
proveedores.read           proveedores.manage
documentos.read            documentos.manage
comunicaciones.read        comunicaciones.send
reclamos.read              reclamos.manage
auditoria.read
```

La autorización evalúa permiso + tenant + scope. Un `consorcista` además debe pasar el vínculo vigente con la UF.

---

## 5. Modelo de datos mínimo

Todas las tablas multi-tenant incluyen `tenant_id UUID NOT NULL`, timestamps e índices que comienzan por `tenant_id`. Las FK de negocio deben impedir referencias cruzadas entre tenants mediante claves compuestas o validación transaccional reforzada por RLS.

### 5.1 Organización

- `consorcios`: tenant, nombre, CUIT, domicilio, tipo, estado, configuración.
- `unidades`: consorcio, código, tipo, superficie, coeficiente, estado.
- `personas`: nombre, documento, email, teléfono. Evitar propietario/inquilino como JSON.
- `unidad_personas`: UF, persona, vínculo (`propietario|inquilino|apoderado`), porcentaje, vigencia desde/hasta.
- `conceptos_expensa`: consorcio o tenant, nombre, categoría y regla de distribución.

Restricciones:

- `UNIQUE(tenant_id, consorcio_id, unidades.codigo)`.
- Coeficiente en `NUMERIC(12,8)`, mayor o igual a cero.
- CUIT normalizado y validado cuando corresponda.
- Vínculos históricos no se sobreescriben: se cierra `valid_to` y se crea otro.

### 5.2 Expensas

- `liquidaciones`: consorcio, período, vencimientos, estado, versión y totales.
- `gastos`: proveedor opcional, comprobante, concepto, importe, fecha, estado.
- `liquidacion_gastos`: snapshot del gasto incluido.
- `liquidacion_items`: importe total por concepto y regla aplicada.
- `liquidacion_unidades`: snapshot de UF, coeficiente y total asignado.
- `liquidacion_unidad_items`: desglose por UF/concepto.

Estados de liquidación:

```text
borrador -> calculada -> confirmada -> publicada -> cerrada
    |           |            |
    +-----------+------------+-> anulada (con reglas según estado)
```

- `borrador`: editable.
- `calculada`: tiene preview reproducible; puede volver a borrador.
- `confirmada`: snapshot inmutable; crea débitos de cuenta corriente.
- `publicada`: documentos disponibles y notificaciones encoladas.
- `cerrada`: no admite operaciones ordinarias del período.
- `anulada`: requiere motivo y asientos compensatorios; nunca borrar movimientos.

La suma de `liquidacion_unidades.total_cents` debe coincidir exactamente con `liquidaciones.total_distribuido_cents`. Los residuos de redondeo se asignan mediante una regla determinista documentada (por defecto, mayor resto y luego código UF).

### 5.3 Cuenta corriente

- `account_entries`: UF, tipo, fecha efectiva, débito, crédito, moneda, referencia y reversa.
- `charges`: deuda originada por una liquidación u otro concepto autorizado.
- `payments`: intención/registro de cobro, canal, importe y estado.
- `payment_allocations`: asigna un pago acreditado a uno o varios cargos.

Invariantes:

- Débito y crédito son no negativos y exactamente uno es mayor a cero.
- Un pago solo impacta saldo al quedar `acreditado`.
- La suma asignada no supera el importe acreditado.
- La asignación no supera el saldo pendiente del cargo, salvo política explícita de saldo a favor.
- Las correcciones crean reversas, no `UPDATE` destructivos.
- El saldo de UF es `SUM(debit_cents-credit_cents)`; si se materializa, debe poder reconstruirse.

### 5.4 Proveedores y documentos

- `proveedores`: CUIT, razón social, contactos, estado.
- `provider_accounts`: CBU/CVU/alias cifrado o protegido; verificación separada.
- `documentos`: owner, tipo, storage key, nombre, MIME, tamaño, SHA-256, estado de antivirus.
- `audit_events`: actor, tenant, acción, recurso, request ID, IP, user-agent y diff seguro.
- `outbox_events`: evento, payload, estado, intentos y próxima ejecución.

Los documentos permanecen privados. La descarga usa URL firmada breve después de autorizar el recurso; nunca aceptar una storage key del cliente.

---

## 6. Casos de uso críticos

### 6.1 Alta e importación

1. `tenant_admin` crea consorcio.
2. Descarga plantilla CSV versionada.
3. Sube archivo; el backend crea un job.
4. Se valida encabezado, tipos, duplicados y referencias sin escribir parcialmente.
5. La UI muestra preview y errores por fila.
6. El usuario confirma; se procesa en transacción o lotes recuperables.
7. Se entrega resumen: creados, actualizados, rechazados y archivo de errores.

No hacer upsert silencioso. El modo `crear`, `actualizar` o `crear_y_actualizar` debe ser explícito.

### 6.2 Liquidar expensas

1. Crear borrador para consorcio/período; no puede existir otro activo para la misma combinación.
2. Agregar gastos y conceptos.
3. Ejecutar cálculo y devolver preview con fórmula y advertencias.
4. Verificar coeficientes, total distribuido, unidades inactivas y gastos sin comprobante.
5. Confirmar usando `expected_version` para evitar edición concurrente.
6. En una transacción: congelar snapshot, generar cargos y outbox.
7. Publicar; el worker genera documentos y envía notificaciones.
8. Un reintento con la misma idempotency key devuelve el resultado anterior.

### 6.3 Registrar un cobro manual

1. Tesorero ingresa fecha, canal, importe, referencia y UF.
2. El backend detecta referencia duplicada y solicita resolución.
3. Se crea pago `pendiente_revision` o `acreditado`, según permiso/política.
4. Se propone asignación FIFO a deuda vencida.
5. El tesorero confirma la asignación.
6. Se crean movimientos y recibo; se registra auditoría.

### 6.4 Pago online futuro

1. El cliente crea una intención con `Idempotency-Key`.
2. Backend calcula el monto; jamás acepta como verdad el monto de la UI.
3. Adaptador PSP crea checkout y devuelve URL/token.
4. La UI muestra `procesando`; nunca `pagado` por retorno del navegador.
5. Webhook autenticado confirma el estado.
6. Evento duplicado retorna 2xx sin duplicar asientos.
7. Un worker consulta operaciones inciertas.
8. Solo `acreditado` genera aplicación y recibo.

### 6.5 Reclamo

Consorcista vinculado crea reclamo con categoría, texto y adjuntos. Administración asigna responsable, responde y cambia estados `abierto -> en_progreso -> resuelto -> cerrado`; la reapertura conserva historial. El consorcista solo ve reclamos propios o de su UF según configuración.

---

## 7. Contrato HTTP

### 7.1 Convenciones

- Base: `/api/v1`.
- JSON en `snake_case`; fechas ISO 8601; importes como enteros `*_cents` y `currency`.
- IDs UUID; no exponer secuencias.
- Paginación por cursor para movimientos/auditoría y por página para catálogos pequeños.
- `X-Request-ID` en petición/respuesta.
- `Idempotency-Key` obligatorio en confirmación, publicación, pagos, asignaciones y webhooks procesados.
- Mutaciones concurrentes reciben `expected_version` o `If-Match`.

Respuesta exitosa:

```json
{
  "data": {},
  "meta": { "request_id": "uuid" }
}
```

Error RFC 9457 (`application/problem+json`):

```json
{
  "type": "https://api.example.com/problems/validation-error",
  "title": "Datos inválidos",
  "status": 422,
  "code": "validation_error",
  "detail": "Hay campos que requieren corrección",
  "instance": "/api/v1/liquidaciones",
  "request_id": "uuid",
  "errors": [{ "field": "period", "code": "already_exists" }]
}
```

### 7.2 Endpoints MVP

```text
POST   /auth/login                    POST   /auth/refresh
POST   /auth/logout                   GET    /me
GET    /memberships                   POST   /auth/select-tenant

GET    /consorcios                    POST   /consorcios
GET    /consorcios/{id}               PATCH  /consorcios/{id}
GET    /consorcios/{id}/unidades      POST   /consorcios/{id}/unidades
POST   /consorcios/{id}/unidades/import-jobs
GET    /import-jobs/{id}              POST   /import-jobs/{id}/confirm

GET    /consorcios/{id}/liquidaciones
POST   /consorcios/{id}/liquidaciones
GET    /liquidaciones/{id}            PATCH  /liquidaciones/{id}
POST   /liquidaciones/{id}/calcular
POST   /liquidaciones/{id}/confirmar
POST   /liquidaciones/{id}/publicar
POST   /liquidaciones/{id}/anular

GET    /consorcios/{id}/gastos        POST   /consorcios/{id}/gastos
GET    /consorcios/{id}/proveedores   POST   /consorcios/{id}/proveedores
GET    /unidades/{id}/cuenta-corriente
GET    /consorcios/{id}/cobranzas     POST   /consorcios/{id}/cobranzas
POST   /cobranzas/{id}/acreditar      POST   /cobranzas/{id}/asignaciones

GET    /documentos/{id}/download-url  POST   /document-upload-intents
GET    /consorcios/{id}/comunicados   POST   /consorcios/{id}/comunicados
POST   /comunicados/{id}/publicar
GET    /consorcios/{id}/reclamos      POST   /consorcios/{id}/reclamos
POST   /reclamos/{id}/mensajes        POST   /reclamos/{id}/transiciones
GET    /audit-events
```

OpenAPI debe describir request, response, errores, permisos e idempotencia. CI debe fallar si el cliente generado o el contrato quedan desactualizados.

---

## 8. Frontend implementable

### 8.1 Navegación MVP

```text
Inicio
Consorcios
  Resumen
  Unidades
  Expensas
  Gastos
  Cobranzas
  Proveedores
  Reclamos
  Comunicados
Administración
  Usuarios y permisos
  Auditoría
Perfil
```

No mostrar módulos futuros como páginas vacías. Ocultarlos con feature flags.

### 8.2 Pantallas obligatorias

| Ruta | Propósito | Estados imprescindibles |
|---|---|---|
| `/login` | Inicio de sesión y selección de tenant | carga, credenciales inválidas, tenant suspendido, MFA |
| `/app` | KPIs con período y consorcio seleccionados | sin consorcios, datos parciales, error |
| `/app/consorcios` | Listar/crear consorcios | vacío, filtros, paginación |
| `/app/consorcios/:id/unidades` | CRUD/importación y vínculos | preview CSV, errores por fila, progreso |
| `/app/consorcios/:id/expensas` | Historial de liquidaciones | filtros por período/estado |
| `/app/liquidaciones/:id` | Wizard y detalle | borrador, calculando, advertencias, conflicto, publicada |
| `/app/consorcios/:id/gastos` | Registro y comprobantes | upload, antivirus pendiente, validación |
| `/app/consorcios/:id/cobranzas` | Cobros y asignaciones | duplicado, no identificado, saldo a favor |
| `/app/unidades/:id/cuenta` | Movimientos y saldo | deuda, crédito, descarga de recibos |
| `/app/consorcios/:id/reclamos` | Cola y conversación | permisos, SLA, adjuntos |
| `/portal` | Inicio del consorcista | varias UFs, deuda, comunicados, reclamos |

### 8.3 Wizard de liquidación

Pasos:

1. Período y vencimientos.
2. Gastos incluidos.
3. Conceptos y reglas de distribución.
4. Cálculo por unidad.
5. Validación: errores bloqueantes y advertencias.
6. Confirmación explícita.
7. Publicación y progreso documental.

La vista debe mostrar total de gastos, total distribuido, diferencia, coeficiente total, unidades alcanzadas y desglose. Confirmar y publicar son acciones distintas. La confirmación exige diálogo con período, importe y cantidad de unidades; la publicación informa canales y destinatarios.

### 8.4 Reglas de interfaz

- Los filtros viven en query params para conservar enlaces y navegación.
- No aplicar actualización optimista a liquidaciones, cobros ni asignaciones.
- Botones financieros se bloquean durante el envío y conservan la misma idempotency key al reintentar.
- Cada página incluye skeleton, vacío útil, error con reintento y estado sin permiso.
- En móvil, tablas de datos se convierten en tarjetas; no depender de scroll horizontal.
- Montos alineados a la derecha, tabulares, formateados con `Intl.NumberFormat`.
- Usar lenguaje de Argentina: `consorcio`, `unidad funcional`, `expensas`, `cobranza`.
- Acciones destructivas requieren consecuencia y motivo cuando corresponda.
- La UI no debe mostrar datos de otro tenant durante cambios de contexto: limpiar caché sensible al cambiar membresía.

---

## 9. Seguridad y privacidad

- Argon2id con parámetros calibrados y límite de intentos de login.
- MFA obligatorio para administradores y tesoreros antes de producción financiera.
- CSRF si se usan cookies; CORS con allowlist exacta.
- CSP estricta, HSTS, protección de framing y MIME sniffing.
- Validación de MIME por contenido, tamaño máximo, antivirus y cuarentena de archivos.
- Cifrado en tránsito y reposo; campos bancarios especialmente protegidos.
- Secrets fuera del repositorio y rotables.
- Logs con email/teléfono/documento enmascarados; jamás tokens, contraseñas ni URLs firmadas.
- Auditoría append-only y exportable.
- Backups cifrados con prueba periódica de restauración.
- Política de retención y eliminación acorde a obligaciones legales y contractuales.
- Para IA futura: no entrenar con datos del cliente, minimizar PII, registrar fuentes y autorización de cada consulta.

### 9.1 Pruebas de aislamiento obligatorias

1. Usuario de tenant A no puede leer un ID conocido de tenant B.
2. Filtros, búsquedas, exports, eventos y descargas respetan tenant y scope.
3. Worker y jobs conservan contexto de tenant.
4. URL firmada solo se emite tras autorizar el documento.
5. RLS bloquea consulta sin `app.tenant_id` y cruce con otro tenant.
6. El cambio de tenant invalida caché y conexiones en tiempo real.

---

## 10. Calidad, pruebas y observabilidad

### 10.1 Backend

- Unitarias para cálculos y máquinas de estados.
- Integración con PostgreSQL real para repositorios, transacciones y RLS.
- Contract tests contra OpenAPI.
- Pruebas de concurrencia: doble confirmación, doble webhook, doble aplicación.
- Property-based tests para distribución y redondeo.
- `go test -race`, lint, análisis de vulnerabilidades y migración arriba/abajo en CI.

### 10.2 Frontend

- Unitarias de formatters y validadores.
- Componentes con Testing Library, incluyendo teclado y errores.
- Integración mediante MSW generado/alineado con OpenAPI.
- E2E: login, alta, importación, liquidación, publicación, cobranza y acceso consorcista.
- Axe en pantallas críticas y viewport 360, 768, 1280 y 1440.

### 10.3 SLO inicial

- Disponibilidad mensual objetivo: 99,5% para MVP.
- API p95 menor a 500 ms en operaciones comunes, excluyendo jobs.
- Tasa de errores 5xx menor a 1%.
- Jobs con estado visible y reintentos acotados; dead-letter inspectable.
- RPO 24 h y RTO 4 h iniciales, documentados como compromiso del MVP.

Cada request lleva `request_id`; cada job conserva `correlation_id`, tenant y actor. Dashboards mínimos: latencia, errores, conexiones BD, tamaño de outbox, jobs fallidos, login fallido y envíos.

---

## 11. Roadmap por entregables verificables

### Fase 0 — Descubrimiento y contrato

Entregar ADRs, mapa de dominios, OpenAPI inicial, ERD, prototipo navegable y matriz de permisos. Resolver proveedor de hosting, email, storage y restricciones legales antes de codificar integraciones.

**Salida:** decisiones sin `TODO` críticos y backlog priorizado.

### Fase 1 — Fundación

Repositorio, Compose, CI, migraciones, identidad, tenants, membresías, RBAC, RLS, auditoría, manejo de errores y shell web.

**Demo:** tenant A y B; prueba automática demuestra aislamiento.

### Fase 2 — Organización

Consorcios, UFs, personas/vínculos, importación CSV, proveedores y documentos.

**Demo:** importar 500 UFs con preview, errores por fila y resultado auditable.

### Fase 3 — Expensas

Gastos, conceptos, cálculo determinista, snapshots, cuenta corriente, PDF y publicación por email simulado.

**Demo:** liquidar un período; suma por UF coincide con total y reintento no duplica cargos.

### Fase 4 — Cobranzas

Registro manual/importación, detección de duplicados, asignación, saldo a favor, recibos y dashboard de morosidad.

**Demo:** pago parcial, pago excedente, reversa y reconstrucción de saldo.

### Fase 5 — Portal y operación

Portal consorcista responsive, comunicaciones, reclamos, notificaciones y observabilidad completa.

**Demo:** consorcista ve solo sus UFs y completa un reclamo de punta a punta.

### Fase 6 — Piloto y hardening

Pruebas de carga, accesibilidad, backup/restore, seguridad, migración real asistida, soporte y documentación operativa.

**Salida:** checklist de go-live firmado y rollback ensayado.

### Fases posteriores

Integración PSP/banco, pagos a proveedores, IA/OCR, laboral, barrios, app nativa, amenities y votaciones. Cada una requiere una especificación y threat model propios.

---

## 12. Definition of Done

Una historia está terminada solo si:

- Tiene criterios de aceptación demostrables.
- La autorización se valida en backend y la UI refleja el permiso.
- Incluye migración y rollback si cambia datos.
- OpenAPI y cliente generado están actualizados.
- Maneja carga, vacío, error, reintento y concurrencia.
- Tiene pruebas proporcionales al riesgo y pasan en CI.
- No introduce secretos, PII en logs ni dependencias vulnerables críticas.
- Es usable con teclado y responsive.
- Emite logs/métricas/auditoría necesarios.
- Incluye documentación de operación cuando corresponde.
- No quedan mocks habilitados en producción.

El porcentaje de cobertura por sí solo no define calidad. Ningún flujo crítico puede quedar sin prueba E2E y prueba de aislamiento.

---

## 13. Instrucción maestra para la IA desarrolladora

Copiar el siguiente bloque como mensaje inicial y adjuntar este documento:

```text
Actuá como líder técnico y desarrollador senior del sistema definido en
"consorcioabierto-especificacion-mejorada-para-ia.md".

Reglas obligatorias:
1. Este documento es la fuente principal. Si los borradores anteriores se contradicen,
   prevalece esta versión.
2. Trabajá una sola fase y una historia pequeña por vez. No implementes módulos futuros.
3. Antes de modificar, inspeccioná el repositorio, AGENTS.md, README, migraciones,
   OpenAPI, tests y estado de Git. No sobrescribas cambios ajenos.
4. Presentá un plan breve con archivos a tocar, invariantes, riesgos y pruebas.
5. Si falta una decisión que modifica datos, seguridad, dinero o alcance, detenete y
   solicitá una decisión. Para detalles reversibles, elegí una opción razonable y documentala.
6. Backend es autoridad. Nunca confíes en tenant_id, permiso, estado o importe enviado
   por el cliente sin resolverlo y validarlo.
7. Toda consulta de negocio debe estar acotada por tenant y scope. Agregá una prueba
   negativa de aislamiento para cada recurso nuevo.
8. Usá BIGINT para centavos, transacciones para invariantes financieras, idempotencia
   para reintentos y asientos compensatorios para reversas.
9. Actualizá primero o junto con el código: migraciones, OpenAPI, cliente y tests.
10. No declares completado algo sin ejecutar verificaciones. Informá comando y resultado.
11. No uses datos ficticios fuera de fixtures/test y no expongas secretos ni PII.
12. Al terminar, resumí: resultado, archivos, decisiones, tests, riesgos y próximo paso.

Primera tarea:
- Si el repositorio está vacío, ejecutá solamente la Fase 0 y proponé el scaffolding.
- Si ya existe código, realizá un gap analysis contra la fase actual antes de implementar.
- No avances a otra fase sin aceptación explícita.
```

### Plantilla para cada tarea

```text
FASE:
HISTORIA:
OBJETIVO DE USUARIO:
ALCANCE INCLUIDO:
FUERA DE ALCANCE:
ARCHIVOS PROBABLES:
REGLAS DE NEGOCIO E INVARIANTES:
PERMISOS Y SCOPE:
CONTRATO API:
CAMBIOS DE DATOS/MIGRACIÓN:
ESTADOS UI:
CRITERIOS DE ACEPTACIÓN (Given/When/Then):
PRUEBAS OBLIGATORIAS:
OBSERVABILIDAD/AUDITORÍA:
COMANDOS DE VERIFICACIÓN:
```

### Formato de entrega esperado de la IA

```text
Resultado
Archivos modificados
Decisiones tomadas
Migraciones y compatibilidad
Pruebas ejecutadas y resultado real
Riesgos o trabajo pendiente
Cómo probar manualmente
Siguiente historia recomendada
```

---

## 14. Criterios de aceptación del MVP completo

1. Dos tenants pueden usar códigos y períodos iguales sin colisión ni filtración.
2. Una administración gestiona múltiples consorcios y limita usuarios por scope.
3. Se importan UFs con validación previa y reporte de errores.
4. Una liquidación confirmada es inmutable y reproducible.
5. El total distribuido coincide exactamente con la suma por UF.
6. Confirmar/publicar dos veces no duplica deuda ni notificaciones.
7. Pagos parciales y excedentes producen saldos correctos.
8. Reversas mantienen historial y reconstrucción contable.
9. El consorcista ve únicamente datos de sus vínculos vigentes.
10. Documentos privados no son enumerables ni descargables sin autorización.
11. Auditoría identifica actor, acción, recurso y request sin secretos.
12. Flujos críticos pasan E2E, aislamiento, accesibilidad y concurrencia en CI.
13. Backup restaurado en entorno de prueba conserva consistencia.
14. El sistema puede operar sin PSP, IA, Redis, Elasticsearch ni Kubernetes.
15. README permite levantar el entorno local con pasos reproducibles.

---

## 15. Decisiones que el dueño del producto debe confirmar

Antes de pasar de Fase 0 a Fase 1, documentar:

- Nombre comercial y dominio; evitar publicar la marca de un tercero como propia.
- Modelo de precios, límites por plan y política de suspensión.
- Si un mismo email puede pertenecer a varias administraciones (esta spec asume que sí).
- Reglas exactas de coeficientes, intereses, segundo vencimiento y redondeo.
- Política de imputación: FIFO, elección manual y saldo a favor.
- Validez legal y formato requerido para liquidaciones, recibos y rendiciones.
- Canales de notificación y quién paga sus costos.
- Proveedor de infraestructura y residencia de datos.
- Retención de documentos, auditoría y datos personales.
- Qué PSP/banco se evaluará y si la plataforma tocará fondos o solo iniciará pagos.
- Datos reales disponibles para migración y responsable de su validación.
- SLA de soporte y objetivos de recuperación.

Hasta confirmar estas decisiones, usar configuraciones explícitas y adaptadores simulados; no codificar supuestos legales o bancarios irreversibles.
