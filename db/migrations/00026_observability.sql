-- +goose Up
-- Funciones globales para métricas de observabilidad ([ADR-0011], §10.3).
-- RLS acota outbox_events por tenant, por eso el conteo va en una función
-- SECURITY DEFINER (owner = consorcio) que ve todas las filas.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.outbox_metrics()
RETURNS TABLE (estado TEXT, total BIGINT)
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
  SELECT o.estado, count(*)::BIGINT AS total
  FROM outbox_events o
  GROUP BY o.estado
$$;
-- +goose StatementEnd

GRANT EXECUTE ON FUNCTION app.outbox_metrics() TO consorcio_app;

-- +goose Down
REVOKE EXECUTE ON FUNCTION app.outbox_metrics() FROM consorcio_app;
DROP FUNCTION IF EXISTS app.outbox_metrics();