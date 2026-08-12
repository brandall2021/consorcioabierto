# ADR-0005 — Identidad global con membresías (email en varias administraciones)

**Estado:** Aceptado
**Fecha:** 12/08/2026
**Confirmado por el owner:** sí (12/08/2026)

## Contexto

Los borradores originales usaban `users.email UNIQUE` global, lo que impedía que una persona participara en más de una administración. La especificación §4.1 corrige esto: identidad global + membresías. La decisión de producto "mismo email en varias admins" fue confirmada.

## Decisión

- `users` es global y su unicidad se aplica sobre `email_normalized` (lowercase + trim) — no sobre el email crudo.
- `tenants` representa cada administración.
- `memberships` = `UNIQUE(user_id, tenant_id)`. Una persona con N administraciones tiene N membresías.
- Roles por membresía (`membership_roles`) y scopes (`role_scopes` con `scope_type = tenant|consorcio|uf`). La PK de roles nunca contiene NULL.
- `platform_admin` es un rol global, separado de la membresía, y no equivale a `tenant_admin`.
- Login devuelve las membresías disponibles; si hay más de una, el usuario elige tenant; solo entonces se emite el access token contextual (claims con `membership_id` y `tenant_id`).
- El `tenant_id` nunca se acepta del cliente: se resuelve desde la membresía activa en el middleware y se cruza con el recurso solicitado.

## Consecuencias

- Un usuario puede ser consorcista en una admin y tesorero en otra, con caché de permisos por membresía.
- La UI limpia caché sensible al cambiar de membresía (regla de interfaz §8.4).
- El flujo login incluye un paso de selección de tenant cuando corresponde.
- Las pruebas de aislamiento cubren también el cruce entre membresías del mismo usuario.
