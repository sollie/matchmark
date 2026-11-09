CREATE TABLE ranges (
    id       INTEGER PRIMARY KEY AUTOINCREMENT,
    name     TEXT NOT NULL COLLATE NOCASE UNIQUE,
    archived INTEGER NOT NULL DEFAULT 0 CHECK (archived IN (0, 1))
);

ALTER TABLE competitions ADD COLUMN range_id INTEGER REFERENCES ranges (id);

ALTER TABLE club_settings ADD COLUMN default_range_id INTEGER REFERENCES ranges (id);
