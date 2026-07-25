package mingshu

const Schema = `
CREATE TABLE IF NOT EXISTS orders (
    id TEXT PRIMARY KEY,
    product TEXT NOT NULL,
    amount INTEGER NOT NULL,
    currency TEXT NOT NULL,
    provider TEXT NOT NULL DEFAULT '',
    email TEXT NOT NULL DEFAULT '',
    locale TEXT NOT NULL DEFAULT 'zh-Hans',
    status TEXT NOT NULL DEFAULT 'pending',
    birth TEXT NOT NULL,
    payment_id TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_lb_orders_email ON orders(email);

CREATE TABLE IF NOT EXISTS reports (
    order_id TEXT PRIMARY KEY REFERENCES orders(id),
    content TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
`
