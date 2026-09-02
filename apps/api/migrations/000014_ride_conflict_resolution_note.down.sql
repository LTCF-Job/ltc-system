DROP INDEX IF EXISTS idx_ride_records_pending_conflict;
ALTER TABLE ride_records DROP COLUMN IF EXISTS conflict_resolution_note;
