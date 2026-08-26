-- Migration: 000001_init_schema.up.sql
-- Description: 初始化長照交通接送系統核心資料表

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "btree_gist";

-- 1. 據點表 (sites)
CREATE TABLE sites (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    address TEXT NOT NULL,
    region TEXT NOT NULL CHECK (region IN ('miaoli', 'hsinchu')),
    open_days SMALLINT[] NOT NULL DEFAULT '{1,2,3,4,5}',
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_site_name_region UNIQUE (name, region)
);

-- 2. 車輛表 (vehicles)
CREATE TABLE vehicles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    plate_no TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL UNIQUE,
    region TEXT NOT NULL CHECK (region IN ('miaoli', 'hsinchu')),
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'maintenance', 'retired')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 3. 司機表 (drivers)
CREATE TABLE drivers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    name_normalized TEXT NOT NULL,
    national_id_cipher BYTEA NOT NULL,
    national_id_hmac BYTEA NOT NULL UNIQUE,
    national_id_masked TEXT NOT NULL,
    email TEXT,
    region TEXT NOT NULL CHECK (region IN ('miaoli', 'hsinchu')),
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'resigned')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 4. 司機車輛指派表 (driver_assignments)
CREATE TABLE driver_assignments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    driver_id UUID NOT NULL REFERENCES drivers(id) ON DELETE RESTRICT,
    vehicle_id UUID NOT NULL REFERENCES vehicles(id) ON DELETE RESTRICT,
    is_primary BOOLEAN NOT NULL DEFAULT true,
    effective_range DATERANGE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT no_overlapping_primary_driver EXCLUDE USING gist (
        vehicle_id WITH =,
        effective_range WITH &&
    ) WHERE (is_primary = true)
);

-- 5. 個案表 (cases)
CREATE TABLE cases (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    name_normalized TEXT NOT NULL,
    national_id_cipher BYTEA NOT NULL,
    national_id_hmac BYTEA NOT NULL UNIQUE,
    national_id_masked TEXT NOT NULL,
    home_address TEXT NOT NULL,
    region TEXT NOT NULL CHECK (region IN ('miaoli', 'hsinchu')),
    ltc_level TEXT,
    service_category SMALLINT NOT NULL DEFAULT 1 CHECK (service_category IN (1, 2)),
    service_usage_type SMALLINT NOT NULL DEFAULT 2 CHECK (service_usage_type IN (1, 2, 3, 4)),
    claim_start_date DATE NOT NULL,
    claim_end_date DATE,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'closed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_claim_date_range CHECK (claim_end_date IS NULL OR claim_end_date >= claim_start_date)
);

-- 6. 個案排班設定 (case_schedules)
CREATE TABLE case_schedules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    case_id UUID NOT NULL REFERENCES cases(id) ON DELETE RESTRICT,
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE RESTRICT,
    effective_range DATERANGE NOT NULL,
    weekdays SMALLINT[] NOT NULL,
    trip_pattern SMALLINT NOT NULL CHECK (trip_pattern IN (1, 2, 4)),
    unit_price NUMERIC(6,2) NOT NULL DEFAULT 115.00 CHECK (unit_price > 0),
    distance_km NUMERIC(5,2) NOT NULL CHECK (distance_km > 0),
    service_duration_min SMALLINT NOT NULL DEFAULT 10 CHECK (service_duration_min BETWEEN 1 AND 240),
    service_code TEXT NOT NULL DEFAULT 'BD03',
    note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT no_overlapping_case_schedule EXCLUDE USING gist (
        case_id WITH =,
        effective_range WITH &&
    )
);

-- 7. 排班趟次時段明細 (schedule_legs)
CREATE TABLE schedule_legs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    schedule_id UUID NOT NULL REFERENCES case_schedules(id) ON DELETE CASCADE,
    leg_seq SMALLINT NOT NULL CHECK (leg_seq BETWEEN 1 AND 4),
    direction TEXT NOT NULL CHECK (direction IN ('outbound', 'inbound')),
    period TEXT NOT NULL CHECK (period IN ('am', 'pm')),
    depart_time TIME NOT NULL,
    arrive_time TIME,
    run_no SMALLINT NOT NULL DEFAULT 1 CHECK (run_no >= 1),
    vehicle_id UUID REFERENCES vehicles(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_schedule_leg_seq UNIQUE (schedule_id, leg_seq)
);

-- 8. Google 表單註冊表 (google_forms)
CREATE TABLE google_forms (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    vehicle_id UUID NOT NULL REFERENCES vehicles(id) ON DELETE RESTRICT,
    title TEXT NOT NULL,
    sheet_id TEXT NOT NULL UNIQUE,
    ingest_secret_ref TEXT NOT NULL,
    last_synced_at TIMESTAMPTZ,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 9. 表單欄位對應表 (form_columns)
CREATE TABLE form_columns (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    form_id UUID NOT NULL REFERENCES google_forms(id) ON DELETE CASCADE,
    column_index INT NOT NULL,
    column_header TEXT NOT NULL,
    cleaned_name TEXT,
    kind TEXT NOT NULL DEFAULT 'unknown' CHECK (kind IN ('meta', 'ride', 'issue', 'unknown')),
    mapping_status TEXT NOT NULL DEFAULT 'pending' CHECK (mapping_status IN ('pending', 'mapped', 'ignored')),
    case_id UUID REFERENCES cases(id) ON DELETE SET NULL,
    leg_seq SMALLINT CHECK (leg_seq BETWEEN 1 AND 4),
    suggested_case_id UUID REFERENCES cases(id) ON DELETE SET NULL,
    suggestion_score NUMERIC(3,2) NOT NULL DEFAULT 0.0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_form_column_idx UNIQUE (form_id, column_index)
);

-- 10. 表單原始回報提交紀錄 (form_submissions)
CREATE TABLE form_submissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    form_id UUID NOT NULL REFERENCES google_forms(id) ON DELETE CASCADE,
    service_date DATE NOT NULL,
    submitted_at TIMESTAMPTZ NOT NULL,
    driver_name_raw TEXT,
    driver_id UUID REFERENCES drivers(id) ON DELETE SET NULL,
    source TEXT NOT NULL DEFAULT 'webhook' CHECK (source IN ('webhook', 'sheets_sync', 'manual')),
    payload JSONB NOT NULL,
    issue_text TEXT,
    anomaly_flags TEXT[],
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_form_submission UNIQUE (form_id, service_date, submitted_at)
);

-- 11. 回報搭乘來源紀錄 (ride_sources)
CREATE TABLE ride_sources (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    submission_id UUID NOT NULL REFERENCES form_submissions(id) ON DELETE CASCADE,
    case_id UUID NOT NULL REFERENCES cases(id) ON DELETE RESTRICT,
    service_date DATE NOT NULL,
    leg_seq SMALLINT NOT NULL CHECK (leg_seq BETWEEN 1 AND 4),
    vehicle_id UUID NOT NULL REFERENCES vehicles(id) ON DELETE RESTRICT,
    driver_id UUID REFERENCES drivers(id) ON DELETE SET NULL,
    reported TEXT NOT NULL CHECK (reported IN ('boarded', 'absent')),
    source_column_index INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_ride_sources_case_date ON ride_sources(case_id, service_date, leg_seq);

-- 12. 搭乘紀錄合併主表 (ride_records)
CREATE TABLE ride_records (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    case_id UUID NOT NULL REFERENCES cases(id) ON DELETE RESTRICT,
    service_date DATE NOT NULL,
    leg_seq SMALLINT NOT NULL CHECK (leg_seq BETWEEN 1 AND 4),
    merged_status TEXT NOT NULL CHECK (merged_status IN ('boarded', 'absent', 'unreported')),
    effective_status TEXT NOT NULL CHECK (effective_status IN ('boarded', 'absent', 'unreported')),
    vehicle_id UUID NOT NULL REFERENCES vehicles(id) ON DELETE RESTRICT,
    driver_id UUID REFERENCES drivers(id) ON DELETE SET NULL,
    has_conflict BOOLEAN NOT NULL DEFAULT false,
    conflict_resolved_at TIMESTAMPTZ,
    conflict_resolved_by UUID,
    depart_time_override TIME,
    duration_min_override SMALLINT CHECK (duration_min_override IS NULL OR duration_min_override BETWEEN 1 AND 240),
    not_claimed_aa09 BOOLEAN NOT NULL DEFAULT false,
    corrected_by UUID,
    corrected_at TIMESTAMPTZ,
    correction_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_ride_record_slot UNIQUE (case_id, service_date, leg_seq),
    CONSTRAINT chk_correction_integrity CHECK (
        (corrected_at IS NULL AND corrected_by IS NULL) OR
        (corrected_at IS NOT NULL AND corrected_by IS NOT NULL)
    )
);
CREATE INDEX idx_ride_records_date_effective ON ride_records(service_date, effective_status);

-- 13. 匯出工作主表 (export_jobs)
CREATE TABLE export_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_type TEXT NOT NULL CHECK (job_type IN ('gov_claim', 'trip_summary', 'hsinchu_schedule', 'maintenance_blank')),
    period_ym CHAR(5) NOT NULL,
    region TEXT NOT NULL CHECK (region IN ('miaoli', 'hsinchu')),
    format TEXT NOT NULL DEFAULT 'xlsx' CHECK (format IN ('xlsx', 'zip')),
    filter_case_ids UUID[],
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'succeeded', 'failed')),
    storage_path TEXT,
    file_checksum TEXT,
    precheck JSONB,
    error_message TEXT,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at TIMESTAMPTZ
);

-- 14. 匯出快照行明細 (export_lines)
CREATE TABLE export_lines (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_id UUID NOT NULL REFERENCES export_jobs(id) ON DELETE CASCADE,
    line_no INT NOT NULL,
    case_id UUID NOT NULL REFERENCES cases(id) ON DELETE RESTRICT,
    national_id_masked TEXT NOT NULL,
    service_date_roc INT NOT NULL,
    raw_payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_export_job_line UNIQUE (job_id, line_no)
);

-- 15. 系統稽核日誌 (audit_log - 只可新增)
CREATE TABLE audit_log (
    id BIGSERIAL PRIMARY KEY,
    actor_id UUID,
    actor_role TEXT,
    action TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id TEXT,
    before_data JSONB,
    after_data JSONB,
    ip_address TEXT,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_audit_log_entity ON audit_log(entity_type, entity_id);
CREATE INDEX idx_audit_log_created_at ON audit_log(created_at);

-- 16. 通知收件人管理表 (notification_recipients)
CREATE TABLE notification_recipients (
    id BIGSERIAL PRIMARY KEY,
    topic TEXT NOT NULL CHECK (topic IN ('missing_report', 'driver_leave', 'month_end', 'export_failed')),
    email TEXT NOT NULL,
    display_name TEXT,
    active BOOLEAN NOT NULL DEFAULT true,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_notification_topic_email UNIQUE (topic, email)
);

-- 17. 國定假日與停駛日表 (holidays)
CREATE TABLE holidays (
    holiday_date DATE PRIMARY KEY,
    name TEXT NOT NULL,
    region TEXT CHECK (region IS NULL OR region IN ('miaoli', 'hsinchu')),
    source TEXT NOT NULL DEFAULT 'manual' CHECK (source IN ('gov_calendar', 'manual')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 18. 出勤紀錄表 (attendance_records)
CREATE TABLE attendance_records (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    driver_id UUID NOT NULL REFERENCES drivers(id) ON DELETE RESTRICT,
    record_date DATE NOT NULL,
    status TEXT NOT NULL DEFAULT 'work' CHECK (status IN ('work', 'leave', 'sick', 'off')),
    note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_driver_attendance_date UNIQUE (driver_id, record_date)
);

-- 19. 車輛維修保養紀錄 (maintenance_logs)
CREATE TABLE maintenance_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    vehicle_id UUID NOT NULL REFERENCES vehicles(id) ON DELETE RESTRICT,
    service_date DATE NOT NULL,
    mileage NUMERIC(8,2) NOT NULL CHECK (mileage >= 0),
    items TEXT NOT NULL,
    vendor TEXT,
    cost NUMERIC(10,2) NOT NULL DEFAULT 0.0 CHECK (cost >= 0),
    receipt_url TEXT,
    note TEXT,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 20. 車輛油資紀錄 (fuel_logs)
CREATE TABLE fuel_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    vehicle_id UUID NOT NULL REFERENCES vehicles(id) ON DELETE RESTRICT,
    driver_id UUID REFERENCES drivers(id) ON DELETE SET NULL,
    fuel_date DATE NOT NULL,
    liters NUMERIC(6,2) NOT NULL CHECK (liters > 0),
    cost NUMERIC(8,2) NOT NULL CHECK (cost > 0),
    receipt_url TEXT,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 21. 應用程式設定 (app_settings)
CREATE TABLE app_settings (
    key TEXT PRIMARY KEY,
    value JSONB NOT NULL,
    description TEXT,
    updated_by UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
