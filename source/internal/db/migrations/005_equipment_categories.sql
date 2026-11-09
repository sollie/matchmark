CREATE TABLE migration_005_guard (
    entries_in_60_48_40_matches_must_be_deleted_and_re_entered_before_upgrading INTEGER NOT NULL CHECK (entries_in_60_48_40_matches_must_be_deleted_and_re_entered_before_upgrading = 0)
);

INSERT INTO migration_005_guard
SELECT COUNT(*) FROM entries e JOIN matches m ON m.id = e.match_id WHERE m.course <> '150';

DROP TABLE migration_005_guard;

CREATE TABLE entries_new (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    match_id   INTEGER NOT NULL REFERENCES matches (id) ON DELETE CASCADE,
    shooter_id INTEGER NOT NULL REFERENCES shooters (id),
    category   TEXT NOT NULL CHECK (category IN (
        'Revolver 1500', 'Pistol 1500', 'Revolver Optical Sight 1500', 'Pistol Optical Sight 1500',
        'Open', 'Distinguished Revolver', 'Distinguished Pistol',
        'Standard Revolver 4,25"', 'Standard Revolver 2,75"', 'Standard Revolver 2,75" 5 Shot',
        'Standard Semi-Automatic Pistol 5,5" Production', 'Standard Semi-Automatic Pistol 5,5"', 'Standard Semi-Automatic Pistol 3,7"',
        'Carbine 1500')),
    lane       INTEGER NOT NULL CHECK (lane >= 1),
    withdrawn  INTEGER NOT NULL DEFAULT 0 CHECK (withdrawn IN (0, 1)),
    UNIQUE (match_id, shooter_id)
);

INSERT INTO entries_new (id, match_id, shooter_id, category, lane, withdrawn)
SELECT id, match_id, shooter_id,
    CASE category
        WHEN 'Revolver' THEN 'Revolver 1500'
        WHEN 'Pistol' THEN 'Pistol 1500'
        WHEN 'Revolver OS' THEN 'Revolver Optical Sight 1500'
        WHEN 'Pistol OS' THEN 'Pistol Optical Sight 1500'
        WHEN 'Carbine' THEN 'Carbine 1500'
    END,
    lane, withdrawn
FROM entries;

DROP TABLE entries;
ALTER TABLE entries_new RENAME TO entries;
CREATE UNIQUE INDEX idx_entries_manned_lane ON entries (match_id, lane) WHERE withdrawn = 0;
