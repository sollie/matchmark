CREATE TABLE screens (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    token_hash   TEXT NOT NULL UNIQUE,
    code         TEXT NOT NULL UNIQUE,
    name         TEXT NOT NULL DEFAULT '',
    approved_at  TEXT,
    match_id     INTEGER REFERENCES matches (id) ON DELETE SET NULL,
    created_at   TEXT NOT NULL,
    last_seen_at TEXT NOT NULL
);

CREATE INDEX idx_screens_match ON screens (match_id);
