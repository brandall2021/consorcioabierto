-- name: GetAppMeta :one
SELECT key, value, updated_at
FROM app_meta
WHERE key = $1;

-- name: SetAppMeta :exec
INSERT INTO app_meta (key, value)
VALUES ($1, $2)
ON CONFLICT (key) DO UPDATE
SET value = excluded.value, updated_at = now();
