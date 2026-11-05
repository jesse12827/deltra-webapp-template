-- Purpose : Create the application schema and set database-level privileges.
-- Run as  : PostgreSQL administrator (postgres), connected to the application's database.
-- Changes : Creates the schema if missing and (re)applies privileges. Safe to run again.
-- Prereq  : 001_roles.sql has run, and the database exists, owned by webapp_owner.
-- Stack   : Web-App Development Stack, Sections 9.1–9.3.

CREATE SCHEMA IF NOT EXISTS webapp AUTHORIZATION webapp_owner;

-- Nothing is created in the public schema.
REVOKE CREATE ON SCHEMA public FROM PUBLIC;

-- Only the application role may connect, besides the owner and administrators.
DO $$
BEGIN
    EXECUTE format('REVOKE ALL ON DATABASE %I FROM PUBLIC', current_database());
    EXECUTE format('GRANT CONNECT ON DATABASE %I TO webapp_app', current_database());
    EXECUTE format('ALTER ROLE webapp_owner IN DATABASE %I SET search_path = webapp', current_database());
    EXECUTE format('ALTER ROLE webapp_app IN DATABASE %I SET search_path = webapp', current_database());
END
$$;

GRANT USAGE ON SCHEMA webapp TO webapp_app;

-- Identity columns draw from sequences owned by webapp_owner.
ALTER DEFAULT PRIVILEGES FOR ROLE webapp_owner IN SCHEMA webapp
    GRANT USAGE ON SEQUENCES TO webapp_app;

-- Table privileges are granted table by table in db/migrations, never by default.
