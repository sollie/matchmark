ALTER TABLE matches ADD COLUMN activated_at TEXT;
UPDATE matches SET activated_at = date WHERE status != 'draft';
