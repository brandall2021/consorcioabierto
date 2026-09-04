-- name: CreateImportJob :one
INSERT INTO import_jobs (tenant_id, consorcio_id, modo, plantilla_version, total_filas, filas, errores)
VALUES (app.current_tenant_id(), sqlc.arg(consorcio_id)::UUID, sqlc.arg(modo)::TEXT,
        sqlc.arg(plantilla_version)::TEXT, sqlc.arg(total_filas)::INTEGER,
        sqlc.arg(filas)::JSONB, sqlc.arg(errores)::JSONB)
RETURNING tenant_id, consorcio_id, id, estado, modo, plantilla_version, total_filas,
          filas, errores, creados, actualizados, rechazados, archivo_errores_url,
          created_at, updated_at;

-- name: GetImportJob :one
SELECT tenant_id, consorcio_id, id, estado, modo, plantilla_version, total_filas,
       filas, errores, creados, actualizados, rechazados, archivo_errores_url,
       created_at, updated_at
FROM import_jobs
WHERE tenant_id = app.current_tenant_id()
  AND id = sqlc.arg(id)::UUID;

-- name: SetImportJobEstado :exec
UPDATE import_jobs
SET estado = sqlc.arg(estado)::TEXT, updated_at = now()
WHERE tenant_id = app.current_tenant_id()
  AND id = sqlc.arg(id)::UUID;

-- name: FinishImportJob :one
UPDATE import_jobs
SET estado = sqlc.arg(estado)::TEXT,
    creados = sqlc.arg(creados)::INTEGER,
    actualizados = sqlc.arg(actualizados)::INTEGER,
    rechazados = sqlc.arg(rechazados)::INTEGER,
    errores = sqlc.arg(errores)::JSONB,
    updated_at = now()
WHERE tenant_id = app.current_tenant_id()
  AND id = sqlc.arg(id)::UUID
RETURNING tenant_id, consorcio_id, id, estado, modo, plantilla_version, total_filas,
          filas, errores, creados, actualizados, rechazados, archivo_errores_url,
          created_at, updated_at;