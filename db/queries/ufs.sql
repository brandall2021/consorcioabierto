-- name: ListUnidades :many
SELECT tenant_id, consorcio_id, id, codigo, tipo, superficie, coeficiente, estado, created_at, updated_at
FROM unidades
WHERE tenant_id = app.current_tenant_id()
  AND consorcio_id = sqlc.arg(consorcio_id)::UUID
  AND (sqlc.arg(estado)::TEXT = '' OR estado = sqlc.arg(estado)::TEXT)
ORDER BY codigo ASC, id ASC;

-- name: GetUnidad :one
SELECT tenant_id, consorcio_id, id, codigo, tipo, superficie, coeficiente, estado, created_at, updated_at
FROM unidades
WHERE tenant_id = app.current_tenant_id()
  AND consorcio_id = sqlc.arg(consorcio_id)::UUID
  AND id = sqlc.arg(id)::UUID;

-- name: CreateUnidad :one
INSERT INTO unidades (tenant_id, consorcio_id, codigo, tipo, superficie, coeficiente)
VALUES (app.current_tenant_id(), sqlc.arg(consorcio_id)::UUID, sqlc.arg(codigo)::TEXT,
        sqlc.arg(tipo)::TEXT, sqlc.narg(superficie)::NUMERIC, sqlc.narg(coeficiente)::NUMERIC)
RETURNING tenant_id, consorcio_id, id, codigo, tipo, superficie, coeficiente, estado, created_at, updated_at;

-- name: UpdateUnidad :one
UPDATE unidades
SET codigo = COALESCE(sqlc.narg(codigo)::TEXT, codigo),
    tipo = COALESCE(sqlc.narg(tipo)::TEXT, tipo),
    superficie = COALESCE(sqlc.narg(superficie)::NUMERIC, superficie),
    coeficiente = COALESCE(sqlc.narg(coeficiente)::NUMERIC, coeficiente),
    estado = COALESCE(sqlc.narg(estado)::TEXT, estado),
    updated_at = now()
WHERE tenant_id = app.current_tenant_id()
  AND consorcio_id = sqlc.arg(consorcio_id)::UUID
  AND id = sqlc.arg(id)::UUID
RETURNING tenant_id, consorcio_id, id, codigo, tipo, superficie, coeficiente, estado, created_at, updated_at;

-- name: GetUnidadByCodigo :one
SELECT tenant_id, consorcio_id, id, codigo, tipo, superficie, coeficiente, estado, created_at, updated_at
FROM unidades
WHERE tenant_id = app.current_tenant_id()
  AND consorcio_id = sqlc.arg(consorcio_id)::UUID
  AND codigo = sqlc.arg(codigo)::TEXT;

-- name: CloseAllVinculosVigentes :execrows
UPDATE unidad_personas
SET valid_to = sqlc.arg(valid_to)::DATE
WHERE tenant_id = app.current_tenant_id()
  AND unidad_id = sqlc.arg(unidad_id)::UUID
  AND valid_to IS NULL;

-- name: GetPersonaByDocumento :one
SELECT tenant_id, id, nombre, documento, email, telefono, created_at, updated_at
FROM personas
WHERE tenant_id = app.current_tenant_id()
  AND documento = sqlc.arg(documento)::TEXT;

-- name: CreatePersona :one
INSERT INTO personas (tenant_id, nombre, documento, email, telefono)
VALUES (app.current_tenant_id(), sqlc.arg(nombre)::TEXT, sqlc.narg(documento)::TEXT,
        sqlc.narg(email)::TEXT, sqlc.narg(telefono)::TEXT)
RETURNING tenant_id, id, nombre, documento, email, telefono, created_at, updated_at;

-- name: ListVinculosVigentes :many
SELECT p.id AS persona_id, p.nombre, p.documento, p.email, p.telefono,
       up.vinculo, up.porcentaje, up.valid_from
FROM unidad_personas up
JOIN personas p ON p.tenant_id = up.tenant_id AND p.id = up.persona_id
WHERE up.tenant_id = app.current_tenant_id()
  AND up.unidad_id = sqlc.arg(unidad_id)::UUID
  AND up.valid_to IS NULL
ORDER BY up.vinculo, p.nombre, p.id;

-- name: CloseVinculoVigente :execrows
UPDATE unidad_personas
SET valid_to = sqlc.arg(valid_to)::DATE
WHERE tenant_id = app.current_tenant_id()
  AND unidad_id = sqlc.arg(unidad_id)::UUID
  AND persona_id = sqlc.arg(persona_id)::UUID
  AND valid_to IS NULL;

-- name: VinculoVigentePorPersona :one
SELECT id
FROM unidad_personas
WHERE tenant_id = app.current_tenant_id()
  AND unidad_id = sqlc.arg(unidad_id)::UUID
  AND persona_id = sqlc.arg(persona_id)::UUID
  AND vinculo = sqlc.arg(vinculo)::TEXT
  AND valid_to IS NULL;

-- name: CreateVinculo :one
INSERT INTO unidad_personas (tenant_id, unidad_id, persona_id, vinculo, porcentaje, valid_from)
VALUES (app.current_tenant_id(), sqlc.arg(unidad_id)::UUID, sqlc.arg(persona_id)::UUID,
        sqlc.arg(vinculo)::TEXT, sqlc.narg(porcentaje)::NUMERIC, sqlc.arg(valid_from)::DATE)
RETURNING id;