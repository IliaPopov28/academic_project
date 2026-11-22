CREATE TYPE order_status AS ENUM ('pending', 'confirmed', 'cancelled');

CREATE TABLE orders (
    id                UUID PRIMARY KEY,
    idempotency_key   TEXT NOT NULL UNIQUE,
    user_id           UUID NOT NULL,
    restaurant_id     UUID NOT NULL,
    status            order_status NOT NULL,
    reservation_id    UUID,
    payment_id        UUID,
    total_price_minor BIGINT NOT NULL CHECK (total_price_minor >= 0),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE order_items (
    order_id     UUID NOT NULL REFERENCES orders (id),
    menu_item_id UUID NOT NULL,
    quantity     INTEGER NOT NULL CHECK (quantity > 0),
    price_minor  BIGINT NOT NULL CHECK (price_minor >= 0),
    PRIMARY KEY (order_id, menu_item_id)
);

CREATE TABLE outbox (
    id           UUID PRIMARY KEY,
    event_type   TEXT NOT NULL,
    payload      BYTEA NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at TIMESTAMPTZ
);

CREATE INDEX outbox_unpublished_idx ON outbox (created_at) WHERE published_at IS NULL;

CREATE TABLE processed_events (
    event_id     UUID PRIMARY KEY,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);