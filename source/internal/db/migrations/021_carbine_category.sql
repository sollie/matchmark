CREATE TABLE stages_backup AS SELECT * FROM stages;
CREATE TABLE stage_events_backup AS SELECT * FROM stage_events;

CREATE TABLE entries_new (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    competition_id INTEGER NOT NULL REFERENCES competitions (id) ON DELETE CASCADE,
    shooter_id     INTEGER NOT NULL REFERENCES shooters (id),
    category       TEXT NOT NULL CHECK (category IN ('Revolver', 'Pistol', 'Revolver OS', 'Pistol OS', 'Carbine')),
    lane           INTEGER NOT NULL CHECK (lane >= 1),
    withdrawn      INTEGER NOT NULL DEFAULT 0 CHECK (withdrawn IN (0, 1)),
    UNIQUE (competition_id, shooter_id)
);

INSERT INTO entries_new (id, competition_id, shooter_id, category, lane, withdrawn)
SELECT id, competition_id, shooter_id, CASE category WHEN 'Rifle' THEN 'Carbine' ELSE category END, lane, withdrawn FROM entries;

DROP TABLE entries;
ALTER TABLE entries_new RENAME TO entries;
CREATE UNIQUE INDEX idx_entries_manned_lane ON entries (competition_id, lane) WHERE withdrawn = 0;

INSERT INTO stages SELECT * FROM stages_backup WHERE id NOT IN (SELECT id FROM stages);
INSERT INTO stage_events SELECT * FROM stage_events_backup WHERE id NOT IN (SELECT id FROM stage_events);
DROP TABLE stages_backup;
DROP TABLE stage_events_backup;
