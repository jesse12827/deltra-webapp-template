-- EXAMPLE DOMAIN. "Items" demonstrates the standard patterns: provenance columns,
-- a date-only field, a row version for concurrent-edit protection, and
-- least-privilege grants. Replace it with the application's own tables.

-- +goose Up

CREATE TABLE items (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    title        text        NOT NULL,
    description  text        NOT NULL DEFAULT '',
    status       text        NOT NULL DEFAULT 'open',
    due_on       date,
    version      integer     NOT NULL DEFAULT 1,
    created_at   timestamptz NOT NULL DEFAULT now(),
    created_by   bigint      NOT NULL REFERENCES users (id),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    updated_by   bigint      NOT NULL REFERENCES users (id),
    CONSTRAINT items_title_length       CHECK (char_length(title) BETWEEN 1 AND 200),
    CONSTRAINT items_description_length CHECK (char_length(description) <= 4000),
    CONSTRAINT items_status_valid       CHECK (status IN ('open', 'closed')),
    CONSTRAINT items_version_positive   CHECK (version > 0)
);
CREATE INDEX items_status_idx ON items (status);
CREATE INDEX items_updated_at_idx ON items (updated_at DESC, id DESC);

-- No DELETE: records are closed, not removed.
GRANT SELECT, INSERT, UPDATE ON items TO webapp_app;

-- +goose Down

DROP TABLE items;
