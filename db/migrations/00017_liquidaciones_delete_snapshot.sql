-- +goose Up
-- Recálculo de liquidaciones: borrador→calculada→borrador→calculada reemplaza
-- el snapshot anterior (DeleteLiquidacion* del dominio). El rol de aplicación
-- necesita DELETE con política de tenant en las tablas de snapshot.

CREATE POLICY liquidacion_gastos_delete ON liquidacion_gastos
    FOR DELETE USING (tenant_id = app.current_tenant_id());
CREATE POLICY liquidacion_items_delete ON liquidacion_items
    FOR DELETE USING (tenant_id = app.current_tenant_id());
CREATE POLICY liquidacion_unidades_delete ON liquidacion_unidades
    FOR DELETE USING (tenant_id = app.current_tenant_id());

GRANT DELETE ON liquidacion_gastos, liquidacion_items, liquidacion_unidades TO consorcio_app;

-- liquidacion_unidad_items se limpia por CASCADE desde liquidacion_unidades;
-- las acciones referenciales no están sujetas a RLS de la tabla hija.

-- +goose Down
REVOKE DELETE ON liquidacion_gastos, liquidacion_items, liquidacion_unidades FROM consorcio_app;

DROP POLICY IF EXISTS liquidacion_gastos_delete ON liquidacion_gastos;
DROP POLICY IF EXISTS liquidacion_items_delete ON liquidacion_items;
DROP POLICY IF EXISTS liquidacion_unidades_delete ON liquidacion_unidades;