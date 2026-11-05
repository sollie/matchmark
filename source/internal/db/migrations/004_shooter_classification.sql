ALTER TABLE shooters ADD COLUMN class_revolver INTEGER CHECK (class_revolver BETWEEN 0 AND 5);
ALTER TABLE shooters ADD COLUMN class_pistol INTEGER CHECK (class_pistol BETWEEN 0 AND 5);
ALTER TABLE shooters ADD COLUMN class_revolver_os INTEGER CHECK (class_revolver_os BETWEEN 0 AND 5);
ALTER TABLE shooters ADD COLUMN class_pistol_os INTEGER CHECK (class_pistol_os BETWEEN 0 AND 5);
ALTER TABLE shooters ADD COLUMN class_carbine INTEGER CHECK (class_carbine BETWEEN 0 AND 5);
ALTER TABLE shooters ADD COLUMN wa1500_synced_at TEXT;
