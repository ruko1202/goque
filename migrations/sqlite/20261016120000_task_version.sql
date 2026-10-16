-- +goose Up
-- +goose StatementBegin
ALTER TABLE goque_task ADD COLUMN version INTEGER NOT NULL DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE goque_task DROP COLUMN version;
-- +goose StatementEnd
