ALTER TABLE competitions ADD COLUMN show_names INTEGER NOT NULL DEFAULT 1 CHECK (show_names IN (0, 1));
ALTER TABLE competitions ADD COLUMN show_clubs INTEGER NOT NULL DEFAULT 1 CHECK (show_clubs IN (0, 1));
ALTER TABLE competitions ADD COLUMN show_countries INTEGER NOT NULL DEFAULT 1 CHECK (show_countries IN (0, 1));
ALTER TABLE competitions ADD COLUMN show_stage_scores INTEGER NOT NULL DEFAULT 1 CHECK (show_stage_scores IN (0, 1));
ALTER TABLE competitions ADD COLUMN show_ring_counts INTEGER NOT NULL DEFAULT 0 CHECK (show_ring_counts IN (0, 1));
