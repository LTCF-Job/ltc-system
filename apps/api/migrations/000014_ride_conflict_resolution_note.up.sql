ALTER TABLE ride_records ADD COLUMN conflict_resolution_note TEXT;

CREATE INDEX idx_ride_records_pending_conflict ON ride_records (service_date, leg_seq)
  WHERE has_conflict = true AND conflict_resolved_at IS NULL;
