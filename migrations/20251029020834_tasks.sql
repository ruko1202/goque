-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';
set time zone 'UTC';

create table task (
    id              uuid        primary key default gen_random_uuid(),
    type            text        not null,
    external_id     text        not null,
    payload         jsonb       not null,
    status          text        not null,
    attempts        int         not null,
    errors          text,
    created_at      timestamptz not null    default now(),
    updated_at      timestamptz,
    next_attempt_at timestamptz not null    default now()
);

create unique index task_type_external_id_idx on task (type, external_id);
create index task_type_status_next_attempt_at_idx on task (type, status, next_attempt_at asc );
create index task_type_status_updated_at_idx on task (type, status, updated_at asc );
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
drop table task;
-- +goose StatementEnd
