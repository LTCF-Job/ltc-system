-- 還原 vehicles.region：能對應到單位的車輛沿用單位區域，其餘填空字串以滿足 NOT NULL。
ALTER TABLE vehicles ADD COLUMN region TEXT NOT NULL DEFAULT '';

UPDATE vehicles v
SET region = s.region
FROM sites s
WHERE v.site_id = s.id;

ALTER TABLE vehicles ALTER COLUMN region DROP DEFAULT;

DROP INDEX IF EXISTS idx_vehicles_site;

ALTER TABLE vehicles DROP CONSTRAINT IF EXISTS ck_vehicle_manufacture_ym;

ALTER TABLE vehicles
    DROP COLUMN wheelchair_accessible,
    DROP COLUMN last_inspection_date,
    DROP COLUMN third_party_insurance_expiry,
    DROP COLUMN passenger_insurance_expiry,
    DROP COLUMN compulsory_insurance_expiry,
    DROP COLUMN manufacture_ym,
    DROP COLUMN model,
    DROP COLUMN brand,
    DROP COLUMN owner_name,
    DROP COLUMN site_id;
