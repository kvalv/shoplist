-- Cart tables
CREATE TABLE IF NOT EXISTS carts(
    id text PRIMARY KEY,
    name text NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL,
    target_store integer NOT NULL,
    inactive boolean NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS items(
    id text PRIMARY KEY,
    cart_id text NOT NULL REFERENCES carts(id) ON DELETE CASCADE,
    text text NOT NULL,
    checked boolean NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME,
    clas_chosen integer,
    UNIQUE (cart_id, text)
);

CREATE TABLE IF NOT EXISTS clas_candidates(
    item_id text NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    idx integer NOT NULL,
    gtm_id text NOT NULL,
    name text NOT NULL,
    price real NOT NULL,
    url text NOT NULL,
    picture text NOT NULL,
    reviews integer NOT NULL,
    stock integer NOT NULL,
    area text,
    shelf text,
    PRIMARY KEY (item_id, idx)
);

-- Cron tables
CREATE TABLE IF NOT EXISTS cron_jobs(
    name text PRIMARY KEY,
    attempt int NOT NULL DEFAULT 0,
    last_error text,
    executed_at timestamp
);

