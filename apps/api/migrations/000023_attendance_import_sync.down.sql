DROP TABLE IF EXISTS attendance_import_conflicts;
ALTER TABLE attendance_records DROP COLUMN IF EXISTS source;
