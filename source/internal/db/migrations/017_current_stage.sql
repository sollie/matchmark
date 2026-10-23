ALTER TABLE competitions ADD COLUMN current_stage INTEGER NOT NULL DEFAULT 1 CHECK (current_stage >= 1);
