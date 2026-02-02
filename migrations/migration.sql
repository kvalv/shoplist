-- Cart tables
CREATE TABLE IF NOT EXISTS carts(
    id text PRIMARY KEY,
    name text NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL,
    created_by text,
    target_store integer NOT NULL,
    inactive boolean NOT NULL DEFAULT FALSE,
    FOREIGN KEY (created_by) REFERENCES users(user_id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS collaborators(
    user_id text NOT NULL,
    cart_id text NOT NULL,
    created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    FOREIGN KEY (cart_id) REFERENCES carts(user_id) ON DELETE CASCADE
);

-- only when creator exists, it is nullable
CREATE TRIGGER IF NOT EXISTS ensure_collaborator
    AFTER INSERT ON carts
    FOR EACH ROW
    WHEN NEW.created_by IS NOT NULL
BEGIN
    INSERT INTO collaborators(user_id,
    cart_id)
VALUES(NEW.created_by,
NEW.id);

END;

-- CREATE TRIGGER IF NOT EXISTS set_created_by
--     AFTER INSERT ON collaborators
--     FOR EACH ROW
--     WHEN NEW.user_id =(
--     SELECT
--         created_by
--     FROM
--         carts
--     WHERE
--         id = NEW.cart_id)
-- BEGIN
--     UPDATE carts SET created_by = NEW.user_id
-- WHERE
--     id = NEW.cart_id;
-- END;
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

-- User tables
CREATE TABLE IF NOT EXISTS users(
    user_id text PRIMARY KEY,
    name text NOT NULL,
    email text NOT NULL,
    picture text NULL,
    created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
);

