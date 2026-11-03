-- name: InsertAuditEvent :exec
INSERT INTO audit_events (user_id, action, record_type, record_id, changes, request_id)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: ListAuditEvents :many
SELECT * FROM audit_events
ORDER BY occurred_at DESC, id DESC
LIMIT $1;

-- name: ListAuditEventsForRecord :many
SELECT * FROM audit_events
WHERE record_type = $1 AND record_id = $2
ORDER BY occurred_at DESC, id DESC;
