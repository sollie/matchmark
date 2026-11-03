ALTER TABLE matches ADD COLUMN group_by_category INTEGER NOT NULL DEFAULT 1 CHECK (group_by_category IN (0, 1));
