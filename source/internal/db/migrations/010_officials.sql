ALTER TABLE users ADD COLUMN waid TEXT;
CREATE UNIQUE INDEX idx_users_waid ON users (waid) WHERE waid IS NOT NULL;

ALTER TABLE matches ADD COLUMN referee_waid TEXT;
ALTER TABLE matches ADD COLUMN referee_name TEXT;
ALTER TABLE matches ADD COLUMN range_officer_waid TEXT;
ALTER TABLE matches ADD COLUMN range_officer_name TEXT;
