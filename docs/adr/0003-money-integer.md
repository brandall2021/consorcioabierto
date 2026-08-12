# ADR-0003 — Dinero como centavos en BIGINT

**Estado:** Aceptado
**Fecha:** 12/08/2026

## Contexto

La especificación (principio 4) prohíbe `float` para dinero. Los totales de liquidación deben coincidir exactamente (criterio de aceptación 5) y las reversas deben reconstruir saldos.

## Decisión

- Todo importe se almacena como `BIGINT` de centavos (`*_cents`) junto a su `currency` (ISO 4217, inicialmente `ARS`).
- Nunca se usan tipos de punto flotante para valores monetarios en ningún punto de la pila (Go, SQL, JSON).
- Las operaciones de suma/resta usan aritmética entera. Los tipos de dominio en Go son `int64` con wrappers tipados por módulo (p. ej. `amount.Cents`).
- El JSON expone `*_cents` como entero; el formateo (incluido el `Intl.NumberFormat` del frontend) es solo presentación.

## Consecuencias

- Comparaciones exactas y sin errores de representación.
- Multiplicación por coeficientes (NUMERIC) requiere redondeo determinista → ver [ADR-0006](./0006-rounding.md).
- Ningún cliente puede enviar importes flotantes; el contrato OpenAPI lo declara como `integer`.

**Alternativas descartadas:** `float`/`double` (inexacto), `NUMERIC` ilimitado en toda la pila (más lento y más propenso a divergencia de precisión en idiomas que no lo soportan nativamente).
