ALTER TABLE cases
    ADD COLUMN household_type TEXT,
    ADD COLUMN gender TEXT,
    ADD COLUMN birth_date DATE,
    ADD COLUMN care_contact_role TEXT,
    ADD COLUMN care_contact_name TEXT,
    ADD COLUMN registered_address TEXT;

CREATE TABLE case_transport_preferences (
    case_id UUID PRIMARY KEY REFERENCES cases(id) ON DELETE CASCADE,
    site_id UUID REFERENCES sites(id) ON DELETE SET NULL,
    outbound_vehicle_id UUID REFERENCES vehicles(id) ON DELETE SET NULL,
    inbound_vehicle_id UUID REFERENCES vehicles(id) ON DELETE SET NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
