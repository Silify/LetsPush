-- What became of a notification sent through the native API, so the sender can ask. It holds no message text.
CREATE TABLE deliveries (
    id         TEXT PRIMARY KEY,
    app_id     INTEGER NOT NULL REFERENCES apps (id) ON DELETE CASCADE,
    key_id     INTEGER NOT NULL REFERENCES access_keys (id) ON DELETE CASCADE,
    status     TEXT NOT NULL,                  -- pending, sent, failed, unregistered
    request_id TEXT NOT NULL DEFAULT '',       -- to find the server's log lines
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE INDEX deliveries_created ON deliveries (created_at);
