CREATE TABLE shooters (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    last_name  TEXT NOT NULL,
    first_name TEXT NOT NULL,
    sex        TEXT NOT NULL CHECK (sex IN ('M', 'F')),
    country    TEXT NOT NULL,
    club       TEXT NOT NULL,
    waid       TEXT
);

CREATE UNIQUE INDEX idx_shooters_waid ON shooters (waid) WHERE waid IS NOT NULL;

CREATE TABLE competitions (
    id     INTEGER PRIMARY KEY AUTOINCREMENT,
    name   TEXT NOT NULL,
    date   TEXT NOT NULL,
    course TEXT NOT NULL CHECK (course IN ('150', '60', '48', '40')),
    status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'active', 'finished'))
);

CREATE TABLE entries (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    competition_id INTEGER NOT NULL REFERENCES competitions (id) ON DELETE CASCADE,
    shooter_id     INTEGER NOT NULL REFERENCES shooters (id),
    category       TEXT NOT NULL CHECK (category IN ('Revolver', 'Pistol', 'Revolver OS', 'Pistol OS', 'Rifle')),
    lane           INTEGER NOT NULL CHECK (lane >= 1),
    withdrawn      INTEGER NOT NULL DEFAULT 0 CHECK (withdrawn IN (0, 1)),
    UNIQUE (competition_id, shooter_id)
);

CREATE UNIQUE INDEX idx_entries_manned_lane ON entries (competition_id, lane) WHERE withdrawn = 0;

CREATE TABLE stages (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    entry_id       INTEGER NOT NULL REFERENCES entries (id) ON DELETE CASCADE,
    stage_number   INTEGER NOT NULL CHECK (stage_number >= 1),
    expected_shots INTEGER NOT NULL CHECK (expected_shots >= 1),
    x              INTEGER NOT NULL DEFAULT 0 CHECK (x >= 0),
    ten            INTEGER NOT NULL DEFAULT 0 CHECK (ten >= 0),
    nine           INTEGER NOT NULL DEFAULT 0 CHECK (nine >= 0),
    eight          INTEGER NOT NULL DEFAULT 0 CHECK (eight >= 0),
    seven          INTEGER NOT NULL DEFAULT 0 CHECK (seven >= 0),
    zero           INTEGER NOT NULL DEFAULT 0 CHECK (zero >= 0),
    miss           INTEGER NOT NULL DEFAULT 0 CHECK (miss >= 0),
    locked         INTEGER NOT NULL DEFAULT 0 CHECK (locked IN (0, 1)),
    scorer_lane    INTEGER,
    scorer_name    TEXT,
    UNIQUE (entry_id, stage_number)
);

CREATE TABLE device_pairings (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    token_hash     TEXT NOT NULL UNIQUE,
    competition_id INTEGER NOT NULL REFERENCES competitions (id) ON DELETE CASCADE,
    lane           INTEGER NOT NULL CHECK (lane >= 1),
    revoked        INTEGER NOT NULL DEFAULT 0 CHECK (revoked IN (0, 1)),
    created_at     TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE admin_sessions (
    token_hash TEXT PRIMARY KEY,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TEXT NOT NULL
);
