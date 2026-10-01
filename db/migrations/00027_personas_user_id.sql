-- +goose Up
-- Vincula una persona con el usuario que la representa en la plataforma.
-- Es el eslabon que faltaba para que el portal del consorcista pueda acotar
-- por vinculo vigente: user -> personas -> unidad_personas -> unidades.
ALTER TABLE personas ADD COLUMN IF NOT EXISTS user_id UUID REFERENCES users(id) ON DELETE SET NULL;

-- Una persona por usuario dentro del tenant. El indice es parcial para
-- permitir varias personas sin usuario (titulares, inquilinos, etc).
CREATE UNIQUE INDEX IF NOT EXISTS personas_user_id_unique
    ON personas(tenant_id, user_id) WHERE user_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS personas_user_id_unique;
ALTER TABLE personas DROP COLUMN IF EXISTS user_id;
