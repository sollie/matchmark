ALTER TABLE sse_tickets ADD COLUMN competition_id INTEGER NOT NULL DEFAULT 0 REFERENCES competitions(id);
