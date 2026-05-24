-- +goose Up
SELECT 'up SQL query';
set time_zone = 'UTC';
-- +goose StatementBegin

CREATE TABLE goque_task (
    id              CHAR(36)     PRIMARY KEY,
    type            VARCHAR(255) NOT NULL,
    external_id     VARCHAR(255) NOT NULL,
    payload         JSON         NOT NULL,
    status          VARCHAR(50)  NOT NULL,
    attempts        INT          NOT NULL,
    errors          TEXT,
    metadata        JSON,
    created_at      TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP    NULL,
    next_attempt_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE UNIQUE INDEX goque_task_type_external_id_idx ON goque_task (type, external_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX goque_task_type_status_next_attempt_at_idx ON goque_task (type, status, next_attempt_at ASC);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX goque_task_type_status_updated_at_idx ON goque_task (type, status, updated_at ASC);
-- +goose StatementEnd

-- +goose Down
SELECT 'down SQL query';
-- +goose StatementBegin
DROP TABLE goque_task;
-- +goose StatementEnd
