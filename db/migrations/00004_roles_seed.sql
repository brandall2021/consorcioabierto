-- +goose Up
INSERT INTO roles (code, label) VALUES
  ('platform_admin', 'Administrador de la plataforma'),
  ('tenant_admin', 'Administrador del tenant'),
  ('consorcio_admin', 'Administrador de consorcio'),
  ('tesorero', 'Tesorero'),
  ('auditor', 'Auditor'),
  ('consorcista', 'Consorcista')
ON CONFLICT (code) DO NOTHING;

-- +goose Down
DELETE FROM roles WHERE code IN ('platform_admin', 'tenant_admin', 'consorcio_admin', 'tesorero', 'auditor', 'consorcista');
