-- +goose Up
-- Auth DML: el rol de la app necesita INSERT/UPDATE sobre sesiones y refresh
-- tokens (login, rotación y logout) manteniendo el aislamiento por usuario.
--
-- El lookup de refresh token por hash no puede depender de app.current_user_id()
-- (aún no se conoce el usuario): se expone una vista propiedad del owner
-- (superuser), que por ser dueño de la tabla bypassa RLS. consorcio_app solo
-- tiene SELECT sobre la vista, por lo que únicamente puede buscar un token si
-- conoce su hash (no enumerable).

CREATE VIEW app.v_refresh_tokens AS
SELECT id, session_id, user_id, token_hash, family_id, expires_at, revoked_at
FROM refresh_tokens;

ALTER TABLE sessions ENABLE ROW LEVEL SECURITY;
CREATE POLICY sessions_insert ON sessions FOR INSERT WITH CHECK (user_id = app.current_user_id());
CREATE POLICY sessions_update ON sessions FOR UPDATE USING (user_id = app.current_user_id());

ALTER TABLE refresh_tokens ENABLE ROW LEVEL SECURITY;
CREATE POLICY refresh_tokens_insert ON refresh_tokens FOR INSERT WITH CHECK (user_id = app.current_user_id());
CREATE POLICY refresh_tokens_update ON refresh_tokens FOR UPDATE USING (user_id = app.current_user_id());

GRANT INSERT, UPDATE ON sessions TO consorcio_app;
GRANT INSERT, UPDATE ON refresh_tokens TO consorcio_app;
GRANT SELECT ON app.v_refresh_tokens TO consorcio_app;

-- +goose Down
REVOKE SELECT ON app.v_refresh_tokens FROM consorcio_app;
REVOKE INSERT, UPDATE ON refresh_tokens FROM consorcio_app;
REVOKE INSERT, UPDATE ON sessions FROM consorcio_app;

DROP POLICY IF EXISTS refresh_tokens_update ON refresh_tokens;
DROP POLICY IF EXISTS refresh_tokens_insert ON refresh_tokens;
DROP POLICY IF EXISTS sessions_update ON sessions;
DROP POLICY IF EXISTS sessions_insert ON sessions;

DROP VIEW IF EXISTS app.v_refresh_tokens;
