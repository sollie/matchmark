ALTER TABLE entries ADD COLUMN classification_at_match INTEGER CHECK (classification_at_match BETWEEN 0 AND 5);
ALTER TABLE entries ADD COLUMN class INTEGER CHECK (class BETWEEN 0 AND 5);
ALTER TABLE entries ADD COLUMN club TEXT;
ALTER TABLE entries ADD COLUMN country TEXT;
ALTER TABLE club_settings ADD COLUMN unclassified_class TEXT NOT NULL DEFAULT 'Master' CHECK (unclassified_class IN ('Master', 'Unclassified'));
