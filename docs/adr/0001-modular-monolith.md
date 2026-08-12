# ADR-0001 — Monolito modular en Go

**Estado:** Aceptado
**Fecha:** 12/08/2026

## Contexto

El producto es un SaaS multi-tenant de tamaño pequeño a mediano. La especificación (v2.0 §1.2) descarta microservicios, Kubernetes, Elasticsearch y event sourcing para el MVP. El equipo es reducido y se necesita velocidad de entrega con verificabilidad contable.

## Decisión

Un único backend Go (binario `apps/api`) que agrupa los módulos de dominio en `internal/{identity,tenancy,consorcios,expensas,cobranzas,proveedores,documentos,comunicaciones,reclamos,platform}`. Cada módulo se divide en `domain`, `application`, `repository` y `transport/http`. El worker (outbox, generación de PDF, envíos) es un segundo binario del mismo repositorio (`apps/worker`) que comparte paquetes `internal`. Un solo despliegue SPA React en `apps/web`.

## Consecuencias

- Un solo esquema de despliegue y una sola base de datos; transacciones ACID entre módulos.
- Riesgo de acoplamiento entre módulos: se controla con límites claros en `domain` y composición en `application`.
- No hay distribución independiente de escalado; se escala replicando el binario si se necesita.
- La frontera de módulos preserva la ruta futura a extraer servicios sin reescribir dominio.

**Alternativas descartadas:** microservicios (costos operativos sin beneficio para el MVP), serverless (transacciones y jobs de larga duración encajan peor).
