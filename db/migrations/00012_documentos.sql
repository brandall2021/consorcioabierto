-- +goose Up
-- Documentos privados (H2.5, §5.4). La storage key la genera el backend; el
-- cliente nunca la envía. El antivirus arranca en 'pendiente' y solo pasa a
-- 'limpio'/'en_cuarentena' tras el escaneo realizado al autorizar la descarga.
-- owner_type/owner_id son polimórficos (gastos, reclamos, UFs) sin FK porque
-- los consumidores llegan en fases posteriores; consorcio_id es opcional y
-- referencia al consorcio dueño del documento cuando el cliente lo declara.

CREATE TABLE IF NOT EXISTS documentos (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    id UUID NOT NULL,
    consorcio_id UUID,
    owner_type TEXT,
    owner_id UUID,
    tipo TEXT NOT NULL,
    nombre TEXT NOT NULL,
    storage_key TEXT NOT NULL,
    mime_type TEXT,
    size_bytes BIGINT NOT NULL,
    sha256 TEXT NOT NULL,
    antivirus TEXT NOT NULL DEFAULT 'pendiente',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, id),
    FOREIGN KEY (tenant_id, consorcio_id) REFERENCES consorcios(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT documentos_nombre_check CHECK (length(nombre) BETWEEN 1 AND 255),
    CONSTRAINT documentos_tipo_check CHECK (length(tipo) BETWEEN 1 AND 50),
    CONSTRAINT documentos_sha256_check CHECK (sha256 ~ '^[0-9a-f]{64}$'),
    CONSTRAINT documentos_size_check CHECK (size_bytes >= 0),
    CONSTRAINT documentos_antivirus_check CHECK (antivirus IN ('pendiente', 'limpio', 'en_cuarentena'))
);

CREATE UNIQUE INDEX IF NOT EXISTS documentos_storage_key_idx ON documentos(tenant_id, storage_key);

ALTER TABLE documentos ENABLE ROW LEVEL SECURITY;
CREATE POLICY documentos_visible ON documentos FOR SELECT USING (tenant_id = app.current_tenant_id());
CREATE POLICY documentos_insert ON documentos FOR INSERT WITH CHECK (tenant_id = app.current_tenant_id());
CREATE POLICY documentos_update ON documentos FOR UPDATE USING (tenant_id = app.current_tenant_id());

GRANT SELECT, INSERT, UPDATE ON documentos TO consorcio_app;

-- +goose Down
REVOKE SELECT, INSERT, UPDATE ON documentos FROM consorcio_app;

DROP POLICY IF EXISTS documentos_update ON documentos;
DROP POLICY IF EXISTS documentos_insert ON documentos;
DROP POLICY IF EXISTS documentos_visible ON documentos;

DROP TABLE IF EXISTS documentos;