CREATE TABLE IF NOT EXISTS scorer_assignments (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    competition_id INTEGER NOT NULL REFERENCES competitions(id) ON DELETE CASCADE,
    shooter_id     INTEGER NOT NULL REFERENCES shooters(id) ON DELETE CASCADE,
    user_id        INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    assigned_at    TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    UNIQUE(competition_id, shooter_id),
    UNIQUE(competition_id, user_id)
);
