# ADR-0006 — Distribución y redondeo por mayor resto, luego código de UF

**Estado:** Aceptado
**Fecha:** 12/08/2026
**Confirmado por el owner:** sí (12/08/2026)

## Contexto

La suma de `liquidacion_unidades.total_cents` debe coincidir exactamente con `liquidaciones.total_distribuido_cents` (criterio de aceptación 5). Al multiplicar importes por coeficientes y repartir por concepto surgen fracciones de centavo que deben asignarse de forma determinista y documentada.

## Decisión

- La distribución se calcula en **centavos con aritmética entera**. Cada concepto se reparte entre las UF en proporción a su coeficiente.
- El residuo (fracción de centavo remanente) se asigna por la regla de **mayor resto** (largest remainder) y, en caso de empate, por **código de UF** ascendente.
- La regla es una función pura y determinista, con property-based tests (toda UF ≥ 0, suma exacta al total, el resultado no depende del orden de entrada salvo el tie-break documentado).
- Los snapshots congelan coeficientes y reglas aplicadas en el momento de confirmar ([ADR-0004](./0004-ledger-and-idempotency.md)); recalcular siempre reproduce el preview.

## Consecuencias

- Los totales siempre cuadran por construcción; la demo de Fase 3 puede verificarlo automáticamente.
- La documentación de la regla queda en el dominio `expensas` y en la UI del wizard (paso cálculo).
- Coeficientes `NUMERIC(12,8) >= 0`; si un consorcio usa coeficiente 0, la UF no recibe reparto (advertencia en validación).

**Alternativas descartadas:** redondeo por cada línea hacia abajo (pierde centavos), asignación al azar (no reproducible).
