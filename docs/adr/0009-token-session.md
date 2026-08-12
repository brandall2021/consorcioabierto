# ADR-0009 — Access token JWT corto + refresh opaco rotativo

**Estado:** Aceptado
**Fecha:** 12/08/2026

## Contexto

La especificación §4.3 define la sesión: access corto, refresh opaco de 30 días, claims mínimos, cookies para web, y revocación de familia ante replay. El backend es la autoridad: los permisos no viajan en el JWT.

## Decisión

- **Access token:** JWT firmado RS256/EdDSA, 10 minutos, claims mínimos `sub, membership_id, tenant_id, session_id, iat, exp, jti`. No incluye listas de permisos.
- **Refresh token:** opaco (32 bytes aleatorios), 30 días, almacenado **hasheado** (SHA-256) en BD junto a su familia/sesión; se rota en cada uso y la versión anterior se invalida.
- **Replay:** si llega un refresh ya consumido, se revoca **toda la familia** (token + sesión).
- **Transporte web:** cookies `HttpOnly`, `Secure`, `SameSite=Lax` + CSRF cuando se usan cookies. Otros clientes usan bearer almacenado en el mecanismo seguro del SO.
- **Permisos:** se resuelven por membresía (consulta + caché) en el middleware de autorización; la UI refleja lo que el backend permite.
- El logout revoca refresh y sesión; los claims llevan `session_id` para invalidación puntual.

## Consecuencias

- Menor superficie de abuso con access corto; el refresh en BD permite revocación explícita.
- Costo: una consulta/caché de permisos por request en vez de trust al JWT.
- El cambio de membresía (select-tenant) emite un token nuevo contextual; la UI limpia caché sensible.

**Alternativas descargadas:** tokens de larga duración (difícil revocar), sesiones stateful sin JWT (pérdida de la capa de presentación), incluir permisos en JWT (stale y pesado).
