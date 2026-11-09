ALTER TABLE entries ADD COLUMN remarks TEXT CHECK (remarks IS NULL OR length(remarks) <= 200);
