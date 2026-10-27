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
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT NOT NULL,
    start_date TEXT NOT NULL,
    end_date   TEXT NOT NULL,
    CHECK (end_date >= start_date)
);

CREATE TABLE matches (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    competition_id    INTEGER NOT NULL REFERENCES competitions (id),
    name              TEXT NOT NULL,
    date              TEXT NOT NULL,
    course            TEXT NOT NULL CHECK (course IN ('150', '60', '48', '40')),
    status            TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'active', 'finished')),
    show_names        INTEGER NOT NULL DEFAULT 1 CHECK (show_names IN (0, 1)),
    show_clubs        INTEGER NOT NULL DEFAULT 1 CHECK (show_clubs IN (0, 1)),
    show_countries    INTEGER NOT NULL DEFAULT 1 CHECK (show_countries IN (0, 1)),
    show_stage_scores INTEGER NOT NULL DEFAULT 1 CHECK (show_stage_scores IN (0, 1)),
    show_ring_counts  INTEGER NOT NULL DEFAULT 0 CHECK (show_ring_counts IN (0, 1)),
    current_stage     INTEGER NOT NULL DEFAULT 1 CHECK (current_stage >= 1)
);

CREATE INDEX idx_matches_competition ON matches (competition_id);

CREATE TABLE entries (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    match_id   INTEGER NOT NULL REFERENCES matches (id) ON DELETE CASCADE,
    shooter_id INTEGER NOT NULL REFERENCES shooters (id),
    category   TEXT NOT NULL CHECK (category IN ('Revolver', 'Pistol', 'Revolver OS', 'Pistol OS', 'Carbine')),
    lane       INTEGER NOT NULL CHECK (lane >= 1),
    withdrawn  INTEGER NOT NULL DEFAULT 0 CHECK (withdrawn IN (0, 1)),
    UNIQUE (match_id, shooter_id)
);

CREATE UNIQUE INDEX idx_entries_manned_lane ON entries (match_id, lane) WHERE withdrawn = 0;

CREATE TABLE stages (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    entry_id       INTEGER NOT NULL REFERENCES entries (id) ON DELETE CASCADE,
    stage_number   INTEGER NOT NULL CHECK (stage_number >= 1),
    expected_shots INTEGER NOT NULL CHECK (expected_shots >= 1),
    x              INTEGER NOT NULL DEFAULT 0 CHECK (x >= 0 AND x <= expected_shots),
    ten            INTEGER NOT NULL DEFAULT 0 CHECK (ten >= 0 AND ten <= expected_shots),
    nine           INTEGER NOT NULL DEFAULT 0 CHECK (nine >= 0 AND nine <= expected_shots),
    eight          INTEGER NOT NULL DEFAULT 0 CHECK (eight >= 0 AND eight <= expected_shots),
    seven          INTEGER NOT NULL DEFAULT 0 CHECK (seven >= 0 AND seven <= expected_shots),
    zero           INTEGER NOT NULL DEFAULT 0 CHECK (zero >= 0 AND zero <= expected_shots),
    miss           INTEGER NOT NULL DEFAULT 0 CHECK (miss >= 0 AND miss <= expected_shots),
    status         TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'awaiting_acceptance', 'accepted')),
    scorer_lane    INTEGER,
    scorer_name    TEXT,
    locked_at      TEXT,
    accepted_by    TEXT,
    accepted_at    TEXT,
    disputed_at    TEXT,
    started        INTEGER NOT NULL DEFAULT 0 CHECK (started IN (0, 1)),
    UNIQUE (entry_id, stage_number)
);

CREATE TABLE stage_events (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    match_id     INTEGER NOT NULL REFERENCES matches (id) ON DELETE CASCADE,
    entry_id     INTEGER NOT NULL REFERENCES entries (id) ON DELETE CASCADE,
    stage_number INTEGER NOT NULL CHECK (stage_number >= 1),
    event        TEXT NOT NULL CHECK (event IN ('locked', 'accepted', 'disputed', 'accepted_all', 'unlocked')),
    actor_type   TEXT NOT NULL CHECK (actor_type IN ('device', 'user')),
    actor_name   TEXT NOT NULL,
    scorer_name  TEXT NOT NULL DEFAULT '',
    shooter_name TEXT NOT NULL DEFAULT '',
    created_at   TEXT NOT NULL
);

CREATE INDEX stage_events_match ON stage_events (match_id, id);

CREATE TABLE device_pairings (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    token_hash TEXT NOT NULL UNIQUE,
    match_id   INTEGER NOT NULL REFERENCES matches (id) ON DELETE CASCADE,
    lane       INTEGER NOT NULL CHECK (lane >= 1),
    revoked    INTEGER NOT NULL DEFAULT 0 CHECK (revoked IN (0, 1)),
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE users (
    id                   INTEGER PRIMARY KEY AUTOINCREMENT,
    username             TEXT NOT NULL UNIQUE COLLATE NOCASE,
    role                 TEXT NOT NULL CHECK (role IN ('admin', 'user')),
    active               INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
    can_download_backup  INTEGER NOT NULL DEFAULT 0 CHECK (can_download_backup IN (0, 1)),
    must_change_password INTEGER NOT NULL DEFAULT 0 CHECK (must_change_password IN (0, 1)),
    password_hash        TEXT NOT NULL,
    created_at           TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at           TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE user_sessions (
    token_hash TEXT PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TEXT NOT NULL
);

CREATE INDEX user_sessions_user_id ON user_sessions (user_id);

CREATE TABLE audit_log (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at TEXT NOT NULL,
    user_id    INTEGER,
    username   TEXT NOT NULL,
    action     TEXT NOT NULL,
    target     TEXT NOT NULL DEFAULT '',
    details    TEXT NOT NULL DEFAULT '',
    ip         TEXT NOT NULL DEFAULT ''
);

CREATE INDEX audit_log_username ON audit_log (username);
CREATE INDEX audit_log_action ON audit_log (action);
CREATE INDEX audit_log_created_at ON audit_log (created_at);

CREATE TRIGGER audit_log_no_update BEFORE UPDATE ON audit_log
BEGIN
    SELECT RAISE(ABORT, 'audit log is append-only');
END;

CREATE TRIGGER audit_log_no_delete BEFORE DELETE ON audit_log
BEGIN
    SELECT RAISE(ABORT, 'audit log is append-only');
END;

CREATE TABLE club_settings (
    id                INTEGER PRIMARY KEY CHECK (id = 1),
    name              TEXT NOT NULL DEFAULT '',
    logo              BLOB,
    logo_content_type TEXT NOT NULL DEFAULT '',
    version           INTEGER NOT NULL DEFAULT 0
);

INSERT INTO club_settings (id) VALUES (1);
