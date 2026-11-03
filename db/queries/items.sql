-- EXAMPLE DOMAIN queries. Search terms arrive already escaped for LIKE
-- (see internal/items). Sort keys are validated against an allow-list in Go.

-- name: CountItems :one
SELECT count(*) FROM items
WHERE (sqlc.arg(search)::text = '' OR title ILIKE '%' || sqlc.arg(search)::text || '%')
  AND (sqlc.arg(status)::text = '' OR status = sqlc.arg(status)::text);

-- name: ListItems :many
SELECT * FROM items
WHERE (sqlc.arg(search)::text = '' OR title ILIKE '%' || sqlc.arg(search)::text || '%')
  AND (sqlc.arg(status)::text = '' OR status = sqlc.arg(status)::text)
ORDER BY
  CASE WHEN sqlc.arg(sort)::text = 'title'  THEN title  END ASC,
  CASE WHEN sqlc.arg(sort)::text = '-title' THEN title  END DESC,
  CASE WHEN sqlc.arg(sort)::text = 'due'    THEN due_on END ASC NULLS LAST,
  CASE WHEN sqlc.arg(sort)::text = '-due'   THEN due_on END DESC NULLS LAST,
  updated_at DESC,
  id DESC
LIMIT sqlc.arg(page_limit)::int OFFSET sqlc.arg(page_offset)::int;

-- name: CountItemsByStatus :many
SELECT status, count(*) AS total FROM items GROUP BY status ORDER BY status;

-- name: GetItem :one
SELECT * FROM items WHERE id = $1;

-- name: CreateItem :one
INSERT INTO items (title, description, status, due_on, created_by, updated_by)
VALUES (sqlc.arg(title), sqlc.arg(description), sqlc.arg(status), sqlc.narg(due_on),
        sqlc.arg(user_id), sqlc.arg(user_id))
RETURNING *;

-- Succeeds only if the row still has the version the user edited (FR concurrent-edit pattern).
-- name: UpdateItem :one
UPDATE items
SET title = sqlc.arg(title),
    description = sqlc.arg(description),
    status = sqlc.arg(status),
    due_on = sqlc.narg(due_on),
    version = version + 1,
    updated_at = now(),
    updated_by = sqlc.arg(user_id)
WHERE id = sqlc.arg(id) AND version = sqlc.arg(version)
RETURNING *;
