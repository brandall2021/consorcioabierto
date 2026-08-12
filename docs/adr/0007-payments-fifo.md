# ADR-0007 — Imputación de cobros FIFO + saldo a favor explícito

**Estado:** Aceptado
**Fecha:** 12/08/2026
**Confirmado por el owner:** sí (12/08/2026)

## Contexto

La especificación §6.3 describe el cobro manual y la sección 15 deja abierta la política de imputación. Se necesita una política por defecto determinista para la demo de Fase 4 (pago parcial, excedente, reversa).

## Decisión

- Un pago solo impacta saldo cuando queda **`acreditado`** (`pendiente_revision` → `acreditado`; el estado `rechazado` no crea movimientos).
- Al acreditar, el backend propone la asignación por **FIFO**: primero la deuda más antigua vencida, en orden de `period`/`due_date`; dentro del mismo cargo, se aplica hasta agotar saldo pendiente.
- La asignación puede ajustarse manualmente por el tesorero dentro del rango permitido:
  - La suma asignada no supera el importe acreditado.
  - No supera el saldo pendiente del cargo, **salvo política explícita de saldo a favor** (excedente), que genera saldo a favor consumible en futuros períodos.
- El pago parcial deja saldo pendiente en el cargo; el excedente deja saldo a favor de la UF.
- La referencia duplicada de un cobro se detecta y solicita resolución (no se acredita silenciosamente).
- Reversas y asignaciones erróneas se corrigen con asientos de reversa, nunca con `UPDATE` destructivos ([ADR-0004](./0004-ledger-and-idempotency.md)).

## Consecuencias

- Comportamiento predecible y comprobable (demo Fase 4: parcial, excedente, reversa, reconstrucción de saldo).
- Existe el concepto de saldo a favor por UF que el frontend debe exponer de forma explícita.
- La política es configurable por tenant en el futuro sin cambiar el modelo de datos.
