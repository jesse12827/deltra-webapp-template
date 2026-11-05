-- name: GetUser :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = lower(sqlc.arg(email)::text);

-- name: ListActiveUsers :many
SELECT * FROM users WHERE active ORDER BY display_name, id;

-- name: CreateUser :one
INSERT INTO users (email, display_name, role, active)
VALUES (lower(sqlc.arg(email)::text), sqlc.arg(display_name), sqlc.arg(role), sqlc.arg(active))
RETURNING *;

-- name: GetUserByEntraID :one
-- Entra ID sign-in matches only on the tenant ID and object ID claims.
SELECT * FROM users WHERE entra_tid = sqlc.arg(entra_tid) AND entra_oid = sqlc.arg(entra_oid);

-- name: SetUserEntraID :exec
-- Replaces any earlier link deliberately. Callers audit the change (auth.LinkUser).
UPDATE users
SET entra_tid = sqlc.arg(entra_tid), entra_oid = sqlc.arg(entra_oid), updated_at = now()
WHERE id = sqlc.arg(id);
