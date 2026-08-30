-- Migration: 000006_multi_driver_per_vehicle.down.sql
-- Description: 還原「一台車同期只有一位主要司機」的限制。
-- is_primary 欄位定義可還原，但原本各列的值已於 up 遷移時遺失，一律回填為 true。

DROP INDEX IF EXISTS idx_driver_assignments_vehicle_id;
ALTER TABLE driver_assignments DROP CONSTRAINT IF EXISTS no_overlapping_driver_assignment;

ALTER TABLE driver_assignments ADD COLUMN is_primary BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE driver_assignments ADD CONSTRAINT no_overlapping_primary_driver EXCLUDE USING gist (
    vehicle_id WITH =,
    effective_range WITH &&
) WHERE (is_primary = true);
