-- Purpose : After restoring a backup into an isolated database, report row counts
--           and check referential integrity so the restore can be compared with the source.
-- Run as  : webapp_owner (or administrator), connected to the RESTORED database.
-- Changes : None. Read-only.
-- Usage   : psql "<restored database>" -f db/scripts/verify_backup.sql

SET search_path = webapp;

SELECT 'users' AS table_name, count(*) AS row_count FROM users
UNION ALL SELECT 'sessions', count(*) FROM sessions
UNION ALL SELECT 'audit_events', count(*) FROM audit_events
UNION ALL SELECT 'items', count(*) FROM items
ORDER BY table_name;

-- Should return zero rows: items whose creator or last editor no longer exists.
SELECT i.id AS orphaned_item_id
FROM items i
LEFT JOIN users c ON c.id = i.created_by
LEFT JOIN users u ON u.id = i.updated_by
WHERE c.id IS NULL OR u.id IS NULL;

-- Applied migrations, to confirm the restored schema version.
SELECT version_id, is_applied, tstamp FROM goose_db_version ORDER BY id DESC LIMIT 5;
