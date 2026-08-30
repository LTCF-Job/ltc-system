-- Migration: 000006_multi_driver_per_vehicle.up.sql
-- Description: 車輛與司機關係改為「一位司機同期只有一台車、一台車同期可有多位司機」。
-- 原本的唯一限制建在車輛上（同期只能有一位主要司機），與實際排班不符；改建在司機上。
-- is_primary 在新規則下失去意義（司機的唯一那台車必然是主要車輛），一併移除。

ALTER TABLE driver_assignments DROP CONSTRAINT IF EXISTS no_overlapping_primary_driver;
ALTER TABLE driver_assignments DROP COLUMN IF EXISTS is_primary;

ALTER TABLE driver_assignments ADD CONSTRAINT no_overlapping_driver_assignment EXCLUDE USING gist (
    driver_id WITH =,
    effective_range WITH &&
);

CREATE INDEX IF NOT EXISTS idx_driver_assignments_vehicle_id ON driver_assignments(vehicle_id);
