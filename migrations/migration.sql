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
    chat_seen_at timestamp,
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    FOREIGN KEY (cart_id) REFERENCES carts(id) ON DELETE CASCADE
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
    id text PRIMARY KEY UNIQUE,
    cart_id text NOT NULL REFERENCES carts(id) ON DELETE CASCADE,
    text text NOT NULL,
    checked boolean NOT NULL,
    discarded boolean NOT NULL DEFAULT FALSE,
    created_at DATETIME NOT NULL,
    created_by text REFERENCES users(user_id) ON DELETE SET NULL,
    updated_by text REFERENCES users(user_id) ON DELETE SET NULL,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    clas_chosen text,
    sort_order integer NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS clas_candidates(
    item_id text NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    idx integer NOT NULL,
    clas_id text NOT NULL DEFAULT '',
    name text NOT NULL,
    price real NOT NULL,
    url text NOT NULL,
    picture text NOT NULL,
    stock integer NOT NULL,
    area text,
    shelf text,
    PRIMARY KEY (item_id, idx),
    UNIQUE (item_id, idx)
);

-- Chat / notes
CREATE TABLE IF NOT EXISTS messages(
    id text PRIMARY KEY,
    cart_id text NOT NULL REFERENCES carts(id) ON DELETE CASCADE,
    role text NOT NULL DEFAULT 'user',
    user_id text REFERENCES users(user_id) ON DELETE SET NULL,
    text text NOT NULL,
    item_id text REFERENCES items(id) ON DELETE SET NULL,
    picture text,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
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
    last_actiom timestamp,
    created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
);

