-- +goose Up
CREATE SCHEMA IF NOT EXISTS app;

CREATE OR REPLACE FUNCTION app.set_app_tenant_id(p UUID) RETURNS void LANGUAGE plpgsql AS $$ BEGIN PERFORM set_config('app.tenant_id', COALESCE(p::text,''), true); END; $$;
CREATE OR REPLACE FUNCTION app.set_app_user_id(p UUID) RETURNS void LANGUAGE plpgsql AS $$ BEGIN PERFORM set_config('app.user_id', COALESCE(p::text,''), true); END; $$;
CREATE OR REPLACE FUNCTION app.current_tenant_id() RETURNS uuid LANGUAGE sql STABLE AS $$ SELECT NULLIF(current_setting('app.tenant_id', true), '')::uuid $$;
CREATE OR REPLACE FUNCTION app.current_user_id() RETURNS uuid LANGUAGE sql STABLE AS $$ SELECT NULLIF(current_setting('app.user_id', true), '')::uuid $$;

ALTER TABLE tenants ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenants_visible ON tenants FOR SELECT USING (id = app.current_tenant_id() OR EXISTS (SELECT 1 FROM memberships m WHERE m.tenant_id = tenants.id AND m.user_id = app.current_user_id()));

ALTER TABLE memberships ENABLE ROW LEVEL SECURITY;
CREATE POLICY memberships_visible ON memberships FOR SELECT USING (user_id = app.current_user_id());

ALTER TABLE membership_roles ENABLE ROW LEVEL SECURITY;
CREATE POLICY membership_roles_visible ON membership_roles FOR SELECT USING (EXISTS (SELECT 1 FROM memberships m WHERE m.id = membership_roles.membership_id AND m.user_id = app.current_user_id()));

ALTER TABLE role_scopes ENABLE ROW LEVEL SECURITY;
CREATE POLICY role_scopes_visible ON role_scopes FOR SELECT USING (EXISTS (SELECT 1 FROM membership_roles mr WHERE mr.id = role_scopes.membership_role_id AND EXISTS (SELECT 1 FROM memberships m WHERE m.id = mr.membership_id AND m.user_id = app.current_user_id())));

ALTER TABLE sessions ENABLE ROW LEVEL SECURITY;
CREATE POLICY sessions_visible ON sessions FOR SELECT USING (user_id = app.current_user_id());

ALTER TABLE refresh_tokens ENABLE ROW LEVEL SECURITY;
CREATE POLICY refresh_tokens_visible ON refresh_tokens FOR SELECT USING (user_id = app.current_user_id());

ALTER TABLE idempotency_keys ENABLE ROW LEVEL SECURITY;
CREATE POLICY idempotency_visible ON idempotency_keys FOR SELECT USING (tenant_id = app.current_tenant_id());
CREATE POLICY idempotency_insert ON idempotency_keys FOR INSERT WITH CHECK (tenant_id = app.current_tenant_id());

DO $$ BEGIN IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname='consorcio_app') THEN CREATE ROLE consorcio_app LOGIN PASSWORD 'consorcio_app_secret'; END IF; END $$;

GRANT USAGE ON SCHEMA public, app TO consorcio_app;
GRANT SELECT ON users, tenants, memberships, roles, membership_roles, role_scopes, sessions, refresh_tokens, idempotency_keys TO consorcio_app;
GRANT INSERT, UPDATE ON idempotency_keys TO consorcio_app;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA app TO consorcio_app;

-- +goose Down
ALTER TABLE idempotency_keys DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS idempotency_visible ON idempotency_keys;
DROP POLICY IF EXISTS idempotency_insert ON idempotency_keys;

ALTER TABLE refresh_tokens DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS refresh_tokens_visible ON refresh_tokens;

ALTER TABLE sessions DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS sessions_visible ON sessions;

ALTER TABLE role_scopes DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS role_scopes_visible ON role_scopes;

ALTER TABLE membership_roles DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS membership_roles_visible ON membership_roles;

ALTER TABLE memberships DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS memberships_visible ON memberships;

ALTER TABLE tenants DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenants_visible ON tenants;

DO $$ BEGIN IF EXISTS (SELECT FROM pg_roles WHERE rolname='consorcio_app') THEN REVOKE ALL PRIVILEGES ON SCHEMA public, app FROM consorcio_app;
DROP ROLE consorcio_app; END IF; END $$;