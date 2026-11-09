ALTER TABLE users ADD COLUMN waid TEXT;
CREATE UNIQUE INDEX idx_users_waid ON users (waid) WHERE waid IS NOT NULL;

CREATE TABLE match_officials (
    id       INTEGER PRIMARY KEY AUTOINCREMENT,
    match_id INTEGER NOT NULL REFERENCES matches (id) ON DELETE CASCADE,
    role     TEXT NOT NULL CHECK (role IN ('referee', 'range_officer')),
    waid     TEXT NOT NULL,
    name     TEXT NOT NULL,
    UNIQUE (match_id, role, waid)
);

CREATE TABLE competition_officials (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    competition_id INTEGER NOT NULL REFERENCES competitions (id) ON DELETE CASCADE,
    role           TEXT NOT NULL CHECK (role IN ('referee', 'range_officer')),
    waid           TEXT NOT NULL,
    name           TEXT NOT NULL,
    UNIQUE (competition_id, role, waid)
);
