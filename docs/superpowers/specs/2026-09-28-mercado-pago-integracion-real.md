# Mercado Pago - integracion real

## Contexto

Ya existe el slice inicial de Mercado Pago sobre cobranzas:
- `POST /api/v1/consorcios/{id}/cobranzas/mercado-pago`
- `POST /api/v1/webhooks/mercado-pago`
- persistencia de `psp_provider`, `psp_preference_id`, `psp_checkout_url`
- tabla `psp_intents` para resolver webhooks por `preference_id`

Hoy el adapter es deterministico. El siguiente paso es reemplazarlo por la integracion real del proveedor sin cambiar el contrato publico ya expuesto.

## Objetivo

Crear la preferencia real de Mercado Pago y procesar el webhook real de pago aprobado, manteniendo el flujo actual de cobranzas y la UI existentes.

## Alcance

Incluye:
- adapter real de Mercado Pago en `internal/cobranzas`
- configuracion por env para credenciales reales
- webhook real con validacion de secreto/firma
- mapeo de estados aprobados a acreditacion idempotente
- pruebas de dominio y HTTP

No incluye:
- redisenar la UI
- pagos a proveedores
- reconciliacion contable adicional
- nuevos estados de negocio visibles al usuario

## Diseño

### 1. Adapter PSP

Reemplazar el adapter deterministico por un adapter real encapsulado en `internal/cobranzas`.

Responsabilidades:
- construir la preferencia con monto, referencia, unidad y callback/webhook
- obtener `checkout_url` y `preference_id`
- devolver un `CheckoutResult` estable para el resto del dominio

El resto del sistema no debe depender del SDK concreto.

### 2. Persistencia

Se conserva el esquema actual:
- `payments.psp_provider`
- `payments.psp_preference_id`
- `payments.psp_checkout_url`
- `psp_intents`

El alta de Mercado Pago sigue guardando la cobranza primero y luego el intento PSP, para que el sistema quede auditable y el webhook pueda correlacionar por `preference_id`.

### 3. Webhook

El handler `POST /api/v1/webhooks/mercado-pago`:
- valida el secreto configurado por env
- lee `preference_id` y `status`
- busca el intento PSP por `preference_id`
- si el estado es `approved`, acredita la cobranza con el flujo de dominio existente
- actualiza el estado del intento PSP

Estados no aprobados se registran como estado del intento y no alteran la cobranza.

### 4. Idempotencia

La acreditacion debe seguir siendo idempotente:
- si el webhook se repite para la misma preferencia, no debe duplicar asientos ni asignaciones
- si la cobranza ya esta acreditada, el webhook responde OK sin cambiar el resultado

### 5. Configuracion

Agregar variables de entorno para produccion:
- `MERCADO_PAGO_ACCESS_TOKEN`
- `MERCADO_PAGO_WEBHOOK_SECRET`

La app debe fallar al arrancar si `PSP_DRIVER=mercadopago` y falta la configuracion minima requerida.

## Flujo de datos

1. El usuario crea una cobranza con Mercado Pago.
2. El backend crea la cobranza y la preferencia del proveedor.
3. El backend persiste los metadatos PSP.
4. Mercado Pago llama al webhook con `preference_id` y `status`.
5. El backend localiza el intento PSP y acredita la cobranza si corresponde.

## Errores

- `401` si el webhook no valida el secreto
- `400` si el payload no tiene `preference_id` o `status`
- `200` si el `preference_id` no existe o el evento ya fue procesado
- `500` solo para fallos internos reales

## Pruebas

Backend:
- caso feliz de creacion de preferencia
- caso feliz de webhook `approved`
- webhook repetido no duplica efectos
- webhook con `preference_id` inexistente no falla

HTTP:
- respuesta `401` por secreto invalido
- respuesta `200` por payload valido

Frontend:
- sin cambios de contrato de UI en este slice

## Criterio de salida

La integracion real queda lista cuando:
- la preferencia se crea con datos reales del proveedor
- el webhook acredita pagos aprobados
- las verificaciones pasan en backend y web
