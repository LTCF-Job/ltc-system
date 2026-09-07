-- 個案批次匯入疑似重複個案：裁決前不建立 cases 資料列，避免半確認的個案
-- 流入排班、司機回報比對與主檔匯出；欄位涵蓋解析出的完整列資料，供裁決頁還原。

CREATE TABLE case_import_duplicate_rows (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    file_hash TEXT NOT NULL,
    row_key TEXT NOT NULL,
    row_index INT NOT NULL,
    sheet_name TEXT,

    name TEXT NOT NULL,
    name_normalized TEXT NOT NULL,
    national_id_cipher BYTEA,
    national_id_hmac BYTEA,
    national_id_masked TEXT,
    household_type TEXT,
    gender TEXT,
    birth_date DATE,
    birth_date_raw TEXT,
    national_id_invalid BOOLEAN NOT NULL DEFAULT false,
    care_contact_role TEXT,
    care_contact_name TEXT,
    registered_address TEXT,
    home_address TEXT,
    region TEXT,
    service_category SMALLINT,
    service_usage_type SMALLINT,
    site_id UUID REFERENCES sites(id) ON DELETE SET NULL,
    site_name_raw TEXT,
    outbound_vehicle_id UUID REFERENCES vehicles(id) ON DELETE SET NULL,
    outbound_vehicle_name_raw TEXT,
    inbound_vehicle_id UUID REFERENCES vehicles(id) ON DELETE SET NULL,
    inbound_vehicle_name_raw TEXT,
    remarks TEXT,

    duplicate_case_id UUID NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'confirmed_new', 'merged_existing')),
    resulting_case_id UUID REFERENCES cases(id) ON DELETE SET NULL,
    resolved_at TIMESTAMPTZ,
    resolved_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_case_import_duplicate_rows_file_row UNIQUE (file_hash, row_key)
);

CREATE INDEX ix_case_import_duplicate_rows_pending ON case_import_duplicate_rows (created_at) WHERE status = 'pending';
CREATE INDEX ix_case_import_duplicate_rows_duplicate_case ON case_import_duplicate_rows (duplicate_case_id);
