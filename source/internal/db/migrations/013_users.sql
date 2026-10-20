DROP TABLE admin_sessions;

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
