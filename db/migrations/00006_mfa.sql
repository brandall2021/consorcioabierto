-- +goose Up
-- MFA (TOTP) y límite de intentos de login.
--
-- login_attempts es append-only y se registra antes de conocer el usuario
-- (el email puede no existir): consorcio_app no recibe SELECT directo, solo
-- EXECUTE sobre funciones SECURITY DEFINER que controlan el acceso.

ALTER TABLE users ADD COLUMN IF NOT EXISTS mfa_secret TEXT;

CREATE TABLE IF NOT EXISTS login_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email_normalized TEXT NOT NULL,
    ip TEXT NOT NULL,
    source TEXT NOT NULL,
    success BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT login_attempts_source_check CHECK (source IN ('password','totp'))
);
CREATE INDEX IF NOT EXISTS login_attempts_email_ip_idx ON login_attempts(email_normalized, ip, created_at);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.record_login_attempt(p_email TEXT, p_ip TEXT, p_source TEXT, p_success BOOLEAN)
RETURNS void LANGUAGE plpgsql SECURITY DEFINER AS $$
BEGIN
    INSERT INTO login_attempts (email_normalized, ip, source, success)
    VALUES (lower(btrim(p_email)), p_ip, p_source, p_success);
END;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION app.count_recent_login_failures(p_email TEXT, p_ip TEXT, p_window INTERVAL)
RETURNS bigint LANGUAGE sql STABLE SECURITY DEFINER AS $$
    SELECT count(*)
    FROM login_attempts
    WHERE email_normalized = lower(btrim(p_email)) AND ip = p_ip AND NOT success
      AND created_at > now() - p_window;
$$;
-- +goose StatementEnd

-- El usuario puede activar/desactivar su propia MFA (UPDATE users).
-- La política de SELECT es abierta (equivalente a users sin RLS en 00003):
-- el lookup por email del login ocurre sin contexto de usuario y así se
-- conserva. El UPDATE sí queda acotado al propio usuario.
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
CREATE POLICY users_visible ON users FOR SELECT USING (TRUE);
CREATE POLICY users_update ON users FOR UPDATE USING (id = app.current_user_id());

GRANT UPDATE ON users TO consorcio_app;
GRANT EXECUTE ON FUNCTION app.record_login_attempt(TEXT, TEXT, TEXT, BOOLEAN) TO consorcio_app;
GRANT EXECUTE ON FUNCTION app.count_recent_login_failures(TEXT, TEXT, INTERVAL) TO consorcio_app;

-- +goose Down
REVOKE EXECUTE ON FUNCTION app.count_recent_login_failures(TEXT, TEXT, INTERVAL) FROM consorcio_app;
REVOKE EXECUTE ON FUNCTION app.record_login_attempt(TEXT, TEXT, TEXT, BOOLEAN) FROM consorcio_app;
REVOKE UPDATE ON users FROM consorcio_app;

DROP POLICY IF EXISTS users_update ON users;
DROP POLICY IF EXISTS users_visible ON users;
ALTER TABLE users DISABLE ROW LEVEL SECURITY;

DROP FUNCTION IF EXISTS app.count_recent_login_failures(TEXT, TEXT, INTERVAL);
DROP FUNCTION IF EXISTS app.record_login_attempt(TEXT, TEXT, TEXT, BOOLEAN);
DROP TABLE IF EXISTS login_attempts;
ALTER TABLE users DROP COLUMN IF EXISTS mfa_secret;
