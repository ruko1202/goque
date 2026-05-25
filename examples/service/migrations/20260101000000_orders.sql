-- +goose Up
-- +goose StatementBegin
-- Domain table used by the transactional-outbox example endpoint
-- (POST /api/orders). The handler INSERTs a row here and enqueues a
-- "send_order_confirmation" goque task within the SAME *sqlx.Tx, so
-- the domain write and the task become durable together — neither
-- can survive without the other.
CREATE TABLE orders (
    id          UUID        PRIMARY KEY,
    customer    TEXT        NOT NULL,
    amount_cents INTEGER    NOT NULL,
    status      TEXT        NOT NULL DEFAULT 'created',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX orders_created_at_idx ON orders (created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE orders;
-- +goose StatementEnd
