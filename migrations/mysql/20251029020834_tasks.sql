-- +goose Up
SELECT 'up SQL query';
set time_zone = 'UTC';
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS task (
    id              CHAR(36)     PRIMARY KEY,
    type            VARCHAR(255) NOT NULL,
    external_id     VARCHAR(255) NOT NULL,
    payload         JSON         NOT NULL,
    status          VARCHAR(50)  NOT NULL,
    attempts        INT          NOT NULL,
    errors          TEXT,
    created_at      TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP    NULL,
    next_attempt_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE UNIQUE INDEX task_type_external_id_idx ON task (type, external_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX task_type_status_next_attempt_at_idx ON task (type, status, next_attempt_at ASC);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX task_type_status_updated_at_idx ON task (type, status, updated_at ASC);
-- +goose StatementEnd

-- +goose Down
SELECT 'down SQL query';
-- +goose StatementBegin
DROP TABLE IF EXISTS task;
-- +goose StatementEnd
