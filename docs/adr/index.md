# ADRs — ConsorcioAbierto

Registro de decisiones de arquitectura. Formato ADR (Michael Nygard): **Estado**, **Contexto**, **Decisión**, **Consecuencias**.

## Índice

| ADR | Decisión |
|---|---|
| [ADR-0001](./0001-modular-monolith.md) | Monolito modular en Go |
| [ADR-0002](./0002-postgres-rls.md) | PostgreSQL 16 con RLS como segunda barrera de aislamiento |
| [ADR-0003](./0003-money-integer.md) | Dinero como centavos en `BIGINT` |
| [ADR-0004](./0004-ledger-and-idempotency.md) | Contabilidad por libro de movimientos + idempotencia |
| [ADR-0005](./0005-global-identity.md) | Identidad global con membresías (email en varias administraciones) |
| [ADR-0006](./0006-rounding.md) | Distribución y redondeo por mayor resto, luego código de UF |
| [ADR-0007](./0007-payments-fifo.md) | Imputación de cobros FIFO + saldo a favor explícito |
| [ADR-0008](./0008-ports-and-adapters.md) | Integraciones desacopladas por puertos y adaptadores |
| [ADR-0009](./0009-token-session.md) | Access token JWT corto + refresh opaco rotativo |
| [ADR-0010](./0010-infra-dokploy.md) | Infraestructura: Dokploy + Postgres + MinIO; sin Redis en el MVP |

## Reglas

- Toda decisión que afecte datos, seguridad o dinero requiere ADR.
- Los ADR son inmutables en su estado "Aceptado"; se actualizan con ADR nuevos o supersession explícita.
