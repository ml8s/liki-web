package naming

const Schema = `
CREATE TABLE IF NOT EXISTS orders (
    order_id        TEXT PRIMARY KEY,
    product         TEXT NOT NULL,
    amount          INTEGER NOT NULL,
    currency        TEXT NOT NULL,
    provider        TEXT NOT NULL DEFAULT '',
    email           TEXT NOT NULL DEFAULT '',
    locale          TEXT NOT NULL DEFAULT 'zh-Hans',
    chat_expires_at TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'pending',
    payment_id      TEXT,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_naming_orders_email ON orders(email, status);
CREATE INDEX IF NOT EXISTS idx_naming_orders_payment ON orders(payment_id) WHERE payment_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS chat_messages (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    order_id   TEXT NOT NULL,
    role       TEXT NOT NULL,
    content    TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_naming_chat_order ON chat_messages(order_id, created_at);

CREATE TABLE IF NOT EXISTS verification_codes (
    code        TEXT PRIMARY KEY,
    status      TEXT NOT NULL DEFAULT 'active',
    order_id    TEXT,
    email       TEXT,
    created_by  TEXT NOT NULL DEFAULT 'admin',
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    used_at     TEXT
);
`
