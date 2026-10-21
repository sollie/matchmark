CREATE TABLE stages_new (
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
    locked         INTEGER NOT NULL DEFAULT 0 CHECK (locked IN (0, 1)),
    scorer_lane    INTEGER,
    scorer_name    TEXT,
    locked_at      TEXT,
    UNIQUE (entry_id, stage_number)
);

INSERT INTO stages_new (id, entry_id, stage_number, expected_shots, x, ten, nine, eight, seven, zero, miss, locked, scorer_lane, scorer_name, locked_at)
SELECT id, entry_id, stage_number, expected_shots, x, ten, nine, eight, seven, zero, miss, locked, scorer_lane, scorer_name, locked_at FROM stages;

DROP TABLE stages;
ALTER TABLE stages_new RENAME TO stages;
