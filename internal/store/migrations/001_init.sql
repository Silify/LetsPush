CREATE TABLE admins (
    id            INTEGER PRIMARY KEY,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at    INTEGER NOT NULL
);

CREATE TABLE apps (
    id                 INTEGER PRIMARY KEY,
    name               TEXT NOT NULL UNIQUE,
    project_id         TEXT NOT NULL DEFAULT '',
    credentials        TEXT NOT NULL DEFAULT '',   -- the Firebase key, sealed
    allow_key_requests INTEGER NOT NULL DEFAULT 0,
    created_at         INTEGER NOT NULL
);

CREATE TABLE access_keys (
    id           INTEGER PRIMARY KEY,
    app_id       INTEGER NOT NULL REFERENCES apps (id) ON DELETE CASCADE,
    key_hash     TEXT NOT NULL UNIQUE,           -- the key itself is shown once and never stored
    prefix       TEXT NOT NULL,                  -- first characters, to recognise a key in lists
    description  TEXT NOT NULL DEFAULT '',
    contact      TEXT NOT NULL DEFAULT '',
    permissions  INTEGER NOT NULL,
    created_at   INTEGER NOT NULL,
    last_used_at INTEGER,
    revoked_at   INTEGER
);

CREATE INDEX access_keys_app ON access_keys (app_id);

-- A device token belongs to the key that registered it.
CREATE TABLE tokens (
    id         INTEGER PRIMARY KEY,
    app_id     INTEGER NOT NULL REFERENCES apps (id) ON DELETE CASCADE,
    key_id     INTEGER NOT NULL REFERENCES access_keys (id) ON DELETE CASCADE,
    token      TEXT NOT NULL,
    device     TEXT NOT NULL,
    channel    TEXT NOT NULL DEFAULT 'default',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    UNIQUE (key_id, token)
);

CREATE INDEX tokens_app ON tokens (app_id);

CREATE TABLE logs (
    id         INTEGER PRIMARY KEY,
    app_id     INTEGER REFERENCES apps (id) ON DELETE CASCADE,
    action     TEXT NOT NULL,
    info       TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL
);

CREATE INDEX logs_created ON logs (created_at);
CREATE INDEX logs_app ON logs (app_id, created_at);
