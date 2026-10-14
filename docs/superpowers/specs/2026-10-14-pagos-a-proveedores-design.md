## Problem Statement

Hoy la plataforma registra proveedores, pero no tiene una vía productiva para iniciar pagos a proveedores de forma auditable, idempotente y ligada al tenant correcto. El objetivo es poder instruir pagos a un proveedor sin tocar fondos ni custodiar dinero dentro de ConsorcioAbierto.

## Solution

Agregar un flujo de pagos a proveedores que permita crear una orden de pago, validarla contra el tenant y el proveedor, persistir su estado y enviarla a un adaptador de pagos/banco para su ejecución. La plataforma solo iniciará pagos y registrará el resultado; no manejará custodia, saldos de cuenta interna ni reconciliación bancaria avanzada en esta fase.

## User Stories

1. As a tesorero, I want to iniciar un pago a un proveedor autorizado, so that I can liquidate una obligación sin salir de la plataforma.
2. As a tesorero, I want to ver el estado de un pago a proveedor, so that I can saber si quedó pendiente, enviado, aprobado o rechazado.
3. As a tesorero, I want to reintentar una instrucción de pago de forma segura, so that duplicated requests do not create duplicate transfers.
4. As a tesorero, I want to usar datos del proveedor ya registrados, so that I can avoid cargar cuentas bancarias repetidas.
5. As a tesorero, I want to registrar un alias/CBU/CVU protegido del proveedor, so that I can instruct transfers without exponer datos sensibles más de lo necesario.
6. As a tesorero, I want to adjuntar una referencia interna al pago, so that I can reconciliarlo con la orden original.
7. As a platform admin, I want to configure the payment provider per environment, so that staging and production can use real drivers without mocks.
8. As an auditor, I want to see quién inició cada pago y cuándo, so that I can trace responsibility end to end.
9. As an auditor, I want to see idempotency keys and provider references, so that I can confirm no duplicados.
10. As an auditor, I want to see the final state and error message of a rejected payment, so that I can diagnose failures.
11. As a consorcio admin, I want payments to remain tenant-scoped, so that one administration never sees another one's provider payments.
12. As a consorcio admin, I want provider payment actions to reuse the existing proveedor data model, so that I do not need a separate registry.
13. As a backend operator, I want the app to fail fast when the payment provider is misconfigured in production, so that we do not ship a broken payment path.
14. As a backend operator, I want payment initiation and webhook processing to be idempotent, so that retries do not duplicate side effects.
15. As a backend operator, I want payment initiation to be auditable before and after provider acknowledgment, so that I can reconstruct the sequence later.
16. As a support user, I want a clear failure reason when the provider rejects a payment, so that I can resolve it without guessing.

## Implementation Decisions

- Reuse the existing tenant-scoped proveedor model as the source of truth for beneficiary identity.
- Introduce a provider-payment domain object with explicit status transitions and immutable audit history.
- Keep payment initiation separate from payment acknowledgment/webhook handling.
- Treat the provider adapter as an interface; the domain should not depend on a specific bank/PSP SDK.
- Persist provider references, idempotency keys, initiator identity, and state transitions.
- Enforce tenant isolation on every query and command.
- Keep the flow “initiate only”: the platform does not hold customer funds or virtual balances.
- Require an idempotency key for every create/submit action that can create provider side effects.
- Use BIGINT centavos for all monetary values.
- Fail fast in production if the real payment adapter is missing required credentials.
- Surface a single canonical status model to the UI and API consumers.
- Allow asynchronous acknowledgment from the provider via webhook or polling, but do not require a UI redesign for the first slice.

## Testing Decisions

- Test external behavior only: API responses, status transitions, and persisted outcomes.
- Cover a happy-path payment initiation test, an idempotency-repeat test, and a tenant-isolation test.
- Cover a provider rejection test and a webhook/acknowledgment test if the provider integration exposes one.
- Cover configuration validation in production for missing credentials.
- Follow the existing style in cobranzas: domain tests around the payment service, HTTP tests around handlers, and config tests around startup validation.

## Out of Scope

- Custody of funds or wallet balances.
- Mass payout batching and scheduling.
- Reconciliation accounting beyond the basic payment status flow.
- UI redesign beyond minimal status rendering.
- Bank statement ingestion or OCR.
- New provider onboarding UX.
- Payouts to employees, vendors outside the provider registry, or cross-tenant transfers.

## Further Notes

This spec intentionally starts from the existing `proveedores` domain and the PSP/banco real groundwork already in place. The first implementation should stay narrow enough to prove the end-to-end flow before expanding to provider payout batching or reconciliation.
