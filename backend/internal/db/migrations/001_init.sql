CREATE TABLE IF NOT EXISTS users (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    username      TEXT    NOT NULL UNIQUE,
    password_hash TEXT    NOT NULL,
    role          TEXT    NOT NULL CHECK(role IN ('user','admin')),
    created_at    TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT    NOT NULL UNIQUE,
    expires_at TEXT    NOT NULL,
    created_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

CREATE TABLE IF NOT EXISTS competitions (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT    NOT NULL,
    date        TEXT    NOT NULL,
    lanes_total INTEGER NOT NULL,
    status      TEXT    NOT NULL DEFAULT 'draft'
                        CHECK(status IN ('draft','active','finished')),
    created_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

CREATE TABLE IF NOT EXISTS shooters (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    competition_id INTEGER NOT NULL REFERENCES competitions(id) ON DELETE CASCADE,
    name           TEXT    NOT NULL,
    wa_id          TEXT    NOT NULL,
    club           TEXT,
    lane_number    INTEGER NOT NULL,
    discipline     TEXT    NOT NULL CHECK(discipline IN ('1500','60','48')),
    weapon_class   TEXT    NOT NULL,
    UNIQUE(competition_id, lane_number)
);

CREATE TABLE IF NOT EXISTS stages (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    shooter_id     INTEGER NOT NULL REFERENCES shooters(id) ON DELETE CASCADE,
    stage_number   INTEGER NOT NULL,
    expected_shots INTEGER NOT NULL,
    fully_locked   INTEGER NOT NULL DEFAULT 0,
    UNIQUE(shooter_id, stage_number)
);

CREATE TABLE IF NOT EXISTS hits (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    stage_id   INTEGER NOT NULL REFERENCES stages(id) ON DELETE CASCADE,
    ring       TEXT    NOT NULL CHECK(ring IN ('X','10','9','8','7','0','M')),
    points     INTEGER NOT NULL,
    created_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

CREATE TABLE IF NOT EXISTS stage_locks (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    stage_id    INTEGER NOT NULL REFERENCES stages(id) ON DELETE CASCADE,
    user_id     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    locked      INTEGER NOT NULL DEFAULT 0,
    locked_time TEXT,
    UNIQUE(stage_id, user_id)
);
