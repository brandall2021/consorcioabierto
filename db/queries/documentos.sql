-- name: InsertDocumento :one
INSERT INTO documentos (tenant_id, id, consorcio_id, owner_type, owner_id, tipo, nombre, storage_key, size_bytes, sha256)
VALUES (app.current_tenant_id(), sqlc.arg(id)::UUID,
        sqlc.narg(consorcio_id)::UUID,
        sqlc.narg(owner_type)::TEXT, sqlc.narg(owner_id)::UUID,
        sqlc.arg(tipo)::TEXT, sqlc.arg(nombre)::TEXT,
        sqlc.arg(storage_key)::TEXT, sqlc.arg(size_bytes)::BIGINT, sqlc.arg(sha256)::TEXT)
RETURNING tenant_id, id, consorcio_id, owner_type, owner_id, tipo, nombre, storage_key, mime_type, size_bytes, sha256, antivirus, created_at, updated_at;

-- name: GetDocumento :one
SELECT tenant_id, id, consorcio_id, owner_type, owner_id, tipo, nombre, storage_key, mime_type, size_bytes, sha256, antivirus, created_at, updated_at
FROM documentos
WHERE tenant_id = app.current_tenant_id()
  AND id = sqlc.arg(id)::UUID;

-- name: UpdateDocumentoScanResult :one
UPDATE documentos
SET antivirus = sqlc.arg(antivirus)::TEXT,
    mime_type = COALESCE(sqlc.narg(mime_type)::TEXT, mime_type),
    updated_at = now()
WHERE tenant_id = app.current_tenant_id()
  AND id = sqlc.arg(id)::UUID
RETURNING tenant_id, id, consorcio_id, owner_type, owner_id, tipo, nombre, storage_key, mime_type, size_bytes, sha256, antivirus, created_at, updated_at;