-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email_normalized = $1;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1;

-- name: UpdateMfaSecret :exec
UPDATE users
SET mfa_secret = $2, updated_at = now()
WHERE id = $1;

-- name: EnableMfa :exec
UPDATE users
SET mfa_enabled = TRUE, updated_at = now()
WHERE id = $1 AND mfa_secret IS NOT NULL;

-- name: DisableMfa :exec
UPDATE users
SET mfa_enabled = FALSE, mfa_secret = NULL, updated_at = now()
WHERE id = $1;

-- name: ListMembershipsForUser :many
SELECT m.id AS membership_id, m.tenant_id, m.status AS membership_status, m.created_at,
       t.name AS tenant_name, t.status AS tenant_status
FROM memberships m
JOIN tenants t ON t.id = m.tenant_id
WHERE m.user_id = $1
ORDER BY m.created_at;

-- name: ListRolesForMembership :many
SELECT r.code, r.label
FROM roles r
JOIN membership_roles mr ON mr.role_id = r.id
WHERE mr.membership_id = $1
ORDER BY r.code;

-- name: ListScopesForMembership :many
SELECT rs.scope_type, rs.scope_id
FROM role_scopes rs
JOIN membership_roles mr ON mr.id = rs.membership_role_id
WHERE mr.membership_id = $1;

-- name: InsertSession :one
INSERT INTO sessions (user_id, membership_id, session_key, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING id;

-- name: GetSessionByID :one
SELECT id, user_id, membership_id, session_key, expires_at, revoked_at, created_at, last_used_at
FROM sessions
WHERE id = $1;

-- name: InsertRefreshToken :exec
INSERT INTO refresh_tokens (session_id, user_id, token_hash, family_id, expires_at)
VALUES ($1, $2, $3, $4, $5);

-- name: LookupRefreshToken :one
SELECT id, session_id, user_id, token_hash, family_id, expires_at, revoked_at
FROM app.v_refresh_tokens
WHERE token_hash = $1;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens SET revoked_at = $2 WHERE id = $1;

-- name: RevokeTokensByFamily :exec
UPDATE refresh_tokens SET revoked_at = $2 WHERE family_id = $1 AND revoked_at IS NULL;

-- name: RevokeSession :exec
UPDATE sessions SET revoked_at = $2 WHERE id = $1;
