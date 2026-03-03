ALTER TABLE refresh_tokens ADD COLUMN family_id TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_family ON refresh_tokens(family_id);
