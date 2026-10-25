-- An app can have a key for each push provider, instead of one Firebase key.
CREATE TABLE app_providers (
    app_id      INTEGER NOT NULL REFERENCES apps (id) ON DELETE CASCADE,
    kind        TEXT NOT NULL,                 -- 'fcm'
    label       TEXT NOT NULL DEFAULT '',      -- shown in lists, for Firebase the project id
    credentials TEXT NOT NULL,                 -- sealed
    updated_at  INTEGER NOT NULL,
    PRIMARY KEY (app_id, kind)
);

INSERT INTO app_providers (app_id, kind, label, credentials, updated_at)
    SELECT id, 'fcm', project_id, credentials, created_at FROM apps WHERE credentials <> '';

ALTER TABLE apps DROP COLUMN project_id;
ALTER TABLE apps DROP COLUMN credentials;
