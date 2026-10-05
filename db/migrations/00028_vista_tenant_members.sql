-- +goose Up
-- Listado de miembros del tenant activo para la administracion de vinculos
-- persona <-> usuario (personas.user_id, migracion 00027).
--
-- Por que una vista: la policy de memberships es user_id = app.current_user_id(),
-- asi que un tenant_admin solo ve su propia membresia y no puede listar a los
-- demas. Es el mismo patrón que app.v_refresh_tokens (00005): la vista es
-- propiedad del owner (superuser) y por eso bypasea RLS, pero el WHERE usa
-- app.current_tenant_id(), de modo que solo se expone lo que ya pertenece al
-- tenant activo. Authorization queda en la capa HTTP: tenant.users.read para
-- listar y tenant.users.manage para vincular.

CREATE VIEW app.v_tenant_members AS
SELECT m.user_id, m.tenant_id, m.status AS membership_status,
       u.email_normalized, u.name, u.status AS user_status
FROM memberships m
JOIN users u ON u.id = m.user_id
WHERE m.tenant_id = app.current_tenant_id();

GRANT SELECT ON app.v_tenant_members TO consorcio_app;

-- +goose Down
REVOKE SELECT ON app.v_tenant_members FROM consorcio_app;
DROP VIEW IF EXISTS app.v_tenant_members;