-- Purpose : Create the application's two database roles.
-- Run as  : PostgreSQL administrator (postgres), connected to any database.
-- Changes : Creates roles if missing. Safe to run again; changes nothing if they exist.
-- Stack   : Web-App Development Stack, Sections 9.2 and 9.3.
--
-- webapp_owner  owns the schema and its objects; used only to run migrations.
-- webapp_app    the running application; receives only the grants made in migrations.
--
-- No passwords are set here. Production authentication uses Unix-socket peer
-- authentication with a pg_ident map (see docs/operations/runbook.md).

DO $$
BEGIN
    CREATE ROLE webapp_owner NOLOGIN;
EXCEPTION WHEN duplicate_object OR unique_violation THEN
    NULL;
END
$$;

DO $$
BEGIN
    CREATE ROLE webapp_app LOGIN;
EXCEPTION WHEN duplicate_object OR unique_violation THEN
    NULL;
END
$$;
