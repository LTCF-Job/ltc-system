-- 司機接送匯報匯入時自動同步司機出勤月曆：比對到司機的匯入資料視為出勤(work)，
-- 當天已有人工登記且狀態不同時不覆蓋，改記一筆待維護的衝突供使用者決定採用哪一筆。

ALTER TABLE attendance_records
    ADD COLUMN source TEXT NOT NULL DEFAULT 'manual' CHECK (source IN ('manual', 'import'));

CREATE TABLE attendance_import_conflicts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    driver_id UUID NOT NULL REFERENCES drivers(id) ON DELETE CASCADE,
    record_date DATE NOT NULL,
    existing_status TEXT NOT NULL CHECK (existing_status IN ('work', 'leave', 'sick', 'off')),
    imported_status TEXT NOT NULL CHECK (imported_status IN ('work', 'leave', 'sick', 'off')),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'resolved')),
    resolved_choice TEXT CHECK (resolved_choice IN ('keep_manual', 'use_import')),
    resolved_by UUID,
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_attendance_import_conflict UNIQUE (driver_id, record_date)
);

CREATE INDEX idx_attendance_import_conflicts_pending ON attendance_import_conflicts(status) WHERE status = 'pending';
