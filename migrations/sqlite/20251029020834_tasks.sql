-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';
CREATE TABLE task (
    id              TEXT        PRIMARY KEY,
    type            TEXT        NOT NULL,
    external_id     TEXT        NOT NULL,
    payload         TEXT        NOT NULL,
    status          TEXT        NOT NULL,
    attempts        INTEGER     NOT NULL,
    errors          TEXT,
    metadata        TEXT,
    created_at      TEXT        NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT,
    next_attempt_at TEXT        NOT NULL DEFAULT (datetime('now'))
);
CREATE UNIQUE INDEX task_type_external_id_idx ON task (type, external_id);
CREATE INDEX task_type_status_next_attempt_at_idx ON task (type, status, next_attempt_at ASC);
CREATE INDEX task_type_status_updated_at_idx ON task (type, status, updated_at ASC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE task;
-- +goose StatementEnd
