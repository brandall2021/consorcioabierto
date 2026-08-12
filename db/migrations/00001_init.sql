-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Tabla de sistema mínima para validar el pipeline migración → sqlc.
CREATE TABLE IF NOT EXISTS app_meta (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS app_meta;
