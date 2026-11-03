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
