DROP TABLE IF EXISTS case_transport_preferences;

ALTER TABLE cases
    DROP COLUMN IF EXISTS registered_address,
    DROP COLUMN IF EXISTS care_contact_name,
    DROP COLUMN IF EXISTS care_contact_role,
    DROP COLUMN IF EXISTS birth_date,
    DROP COLUMN IF EXISTS gender,
    DROP COLUMN IF EXISTS household_type;
