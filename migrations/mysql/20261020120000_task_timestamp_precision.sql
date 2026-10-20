-- +goose Up
-- Store task timestamps with microsecond precision. A plain TIMESTAMP rounds
-- the fraction to the nearest second, so a task enqueued at 12:00:00.6 was
-- stored as 12:00:01 and could not be fetched until then.
-- +goose StatementBegin
ALTER TABLE goque_task
    MODIFY COLUMN created_at      TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    MODIFY COLUMN updated_at      TIMESTAMP(6) NULL,
    MODIFY COLUMN next_attempt_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6);
-- +goose StatementEnd

-- +goose Down
-- Rounds the stored fractions back to whole seconds.
-- +goose StatementBegin
ALTER TABLE goque_task
    MODIFY COLUMN created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    MODIFY COLUMN updated_at      TIMESTAMP NULL,
    MODIFY COLUMN next_attempt_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;
-- +goose StatementEnd
