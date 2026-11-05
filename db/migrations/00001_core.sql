-- Core tables every Deltra web application needs: users, sessions, audit events.
-- Tables are created unqualified; the migration connection's search_path is the
-- application schema (Web-App Development Stack, Section 9).

-- +goose Up

CREATE TABLE users (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email         text        NOT NULL,
    display_name  text        NOT NULL,
    role          text        NOT NULL,
    active        boolean     NOT NULL DEFAULT true,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT users_email_lowercase CHECK (email = lower(email)),
    CONSTRAINT users_email_length    CHECK (char_length(email) BETWEEN 3 AND 254),
    CONSTRAINT users_name_length     CHECK (char_length(display_name) BETWEEN 1 AND 200),
    CONSTRAINT users_role_valid      CHECK (role IN ('admin', 'tech')),
    CONSTRAINT users_email_key       UNIQUE (email)
);

-- Server-side sessions. Only a SHA-256 hash of the session token is stored.
CREATE TABLE sessions (
    token_hash    bytea       PRIMARY KEY,
    user_id       bigint      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at    timestamptz NOT NULL DEFAULT now(),
    last_seen_at  timestamptz NOT NULL DEFAULT now(),
    expires_at    timestamptz NOT NULL,
    CONSTRAINT sessions_hash_length CHECK (octet_length(token_hash) = 32)
);
CREATE INDEX sessions_user_id_idx ON sessions (user_id);
CREATE INDEX sessions_expires_at_idx ON sessions (expires_at);

-- Audit events are append-only: the application may insert and read, never change.
CREATE TABLE audit_events (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    occurred_at  timestamptz NOT NULL DEFAULT now(),
    user_id      bigint      REFERENCES users (id),
    action       text        NOT NULL,
    record_type  text        NOT NULL DEFAULT '',
    record_id    bigint,
    changes      jsonb,
    request_id   text        NOT NULL DEFAULT ''
);
CREATE INDEX audit_events_occurred_at_idx ON audit_events (occurred_at DESC);
CREATE INDEX audit_events_record_idx ON audit_events (record_type, record_id);

GRANT SELECT, INSERT, UPDATE         ON users        TO webapp_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON sessions     TO webapp_app;
GRANT SELECT, INSERT                 ON audit_events TO webapp_app;

-- +goose Down

DROP TABLE audit_events;
DROP TABLE sessions;
DROP TABLE users;
