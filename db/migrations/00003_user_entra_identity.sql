-- Users are matched to their Microsoft Entra ID account on the immutable
-- tenant ID (tid) and object ID (oid) claims, never on email or
-- preferred_username, which directory administrators can change. This rule
-- comes from the Phase 0 security review (2026-11-05, finding 3); Stack
-- Section 10.1 requires only Entra ID sign-in. Both columns are empty for
-- users who have no Entra ID account, such as synthetic development users;
-- such users cannot sign in through Entra ID.

-- +goose Up

ALTER TABLE users
    ADD COLUMN entra_tid uuid,
    ADD COLUMN entra_oid uuid,
    ADD CONSTRAINT users_entra_id_pair CHECK ((entra_tid IS NULL) = (entra_oid IS NULL)),
    ADD CONSTRAINT users_entra_id_key  UNIQUE (entra_tid, entra_oid);

-- +goose Down

ALTER TABLE users
    DROP CONSTRAINT users_entra_id_key,
    DROP CONSTRAINT users_entra_id_pair,
    DROP COLUMN entra_oid,
    DROP COLUMN entra_tid;
