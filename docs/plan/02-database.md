# MatchMark — Database Schema

## File: `backend/internal/db/migrations/001_init.sql`

```sql
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
```

## Design Notes

### hits — individual rows, not aggregated
Each shot is stored as a separate row with its ring label (`X`, `10`, ... `M`) and
numeric point value. When a scorer submits a stage, all existing hits for that stage
are deleted and replaced in a single transaction (PUT semantics). This preserves full
shot-by-shot history within a session and makes X-count derivation trivial
(`SELECT COUNT(*) FROM hits WHERE stage_id=? AND ring='X'`).

### stages — created eagerly on shooter add
When a shooter is added to a competition, the backend immediately creates all stage
rows for that shooter (derived from their discipline via the rulebook package). This
simplifies the scoring API: stage rows always exist; no lazy-creation ambiguity.

### stage_locks — one scorer per lane
Since there is exactly one scorer per lane, `fully_locked` on `stages` is set to `1`
the moment the single scorer's `stage_locks.locked` becomes `1`. Unlock resets both.

### refresh_tokens — hashed storage
The raw refresh token is a 32-byte random value returned to the client. Only its
`SHA-256` hash is stored in the DB. On refresh, the backend hashes the incoming token
and looks it up. Every refresh rotates: old hash deleted, new hash inserted.

### competition.status lifecycle
- `draft`    — visible to admin only; scorers cannot select it
- `active`   — visible to all; scorers can select and score it
- `finished` — read-only for all; export still available to admin

### weapon_class stored as string
The weapon class is stored verbatim (e.g. `Standard Revolver 2.75" 5-Shot`). The
rulebook package in the backend validates the value at write time. No foreign key to
a weapon_classes table — the rulebook is code, not data, since it changes only with
the rulebook edition.
