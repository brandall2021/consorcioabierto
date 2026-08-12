# ADR-0004 — Contabilidad por libro de movimientos + idempotencia

**Estado:** Aceptado
**Fecha:** 12/08/2026

## Contexto

El saldo no se edita manualmente (principio 3). Los criterios de aceptación exigen: confirmar/publicar dos veces no duplica deuda ni notificaciones (6), reversas mantienen historial (8), pagos parciales y excedentes dan saldos correctos (7).

## Decisión

- **Libro de movimientos:** el saldo de una UF se deriva de `account_entries` como `SUM(debit_cents - credit_cents)`. No existe columna de saldo mutable. Las correcciones crean asientos de reversa (`reverses_of`) apuntando al asiento original; jamás `UPDATE` destructivos sobre movimientos.
- **Cargos (charges):** una liquidación confirmada genera cargos por UF (deuda). Los pagos acreditados se asignan a cargos (`payment_allocations`).
- **Idempotencia:** toda operación financiera (confirmar, publicar, cobranzas, asignaciones, webhooks) exige `Idempotency-Key`. El resultado de la primera ejecución se almacena; un reintento con la misma clave devuelve el resultado previo sin re-ejecutar efectos.
- **Versión optimista:** confirmación/publicación envían `expected_version` (o `If-Match`) para detectar edición concurrente de un borrador.
- **Transacciones:** confirmar/publicar ejecutan en una transacción: congelar snapshot → generar cargos → encolar eventos en `outbox_events`.

## Consecuencias

- Los saldos siempre son reconstruibles (requisito de auditoría y de la demo de Fase 4).
- Duplicación imposible por diseño: clave idempotente única + transacción + outbox.
- Complejidad de código algo mayor (helpers de idempotencia y reversas), amortizada por la verificabilidad.
- Las tablas de idempotencia se limpian por política de retención definida en producto.
