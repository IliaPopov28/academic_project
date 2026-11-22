CREATE TABLE restaurants (
    id         UUID PRIMARY KEY,
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE menu_items (
    id            UUID PRIMARY KEY,
    restaurant_id UUID NOT NULL REFERENCES restaurants (id),
    name          TEXT NOT NULL,
    price_minor   BIGINT NOT NULL CHECK (price_minor >= 0),
    available_qty INTEGER NOT NULL CHECK (available_qty >= 0),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX menu_items_restaurant_id_idx ON menu_items (restaurant_id);

CREATE TABLE reservations (
    id                     UUID PRIMARY KEY,
    idempotency_key        TEXT NOT NULL UNIQUE,
    restaurant_id          UUID NOT NULL REFERENCES restaurants (id),
    release_idempotency_key TEXT UNIQUE,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    released_at            TIMESTAMPTZ,
    CHECK (released_at IS NULL OR released_at >= created_at)
);

CREATE TABLE reservation_items (
    reservation_id UUID NOT NULL REFERENCES reservations (id),
    menu_item_id   UUID NOT NULL REFERENCES menu_items (id),
    quantity       INTEGER NOT NULL CHECK (quantity > 0),
    PRIMARY KEY (reservation_id, menu_item_id)
);

CREATE TABLE outbox (
    id           UUID PRIMARY KEY,
    event_type   TEXT NOT NULL,
    payload      BYTEA NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at TIMESTAMPTZ
);

-- Частичный индекс: в него попадают только ещё не отправленные события.
CREATE INDEX outbox_unpublished_idx ON outbox (created_at) WHERE published_at IS NULL;