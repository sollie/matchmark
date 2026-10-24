CREATE TABLE club_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    name TEXT NOT NULL DEFAULT '',
    logo BLOB,
    logo_content_type TEXT NOT NULL DEFAULT '',
    version INTEGER NOT NULL DEFAULT 0
);

INSERT INTO club_settings (id) VALUES (1);
