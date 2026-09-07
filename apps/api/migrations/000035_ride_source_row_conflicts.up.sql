-- 「同一台車同一個案」逐列比對後，值與既有資料不同時暫存待使用者裁決，取代舊的整段
-- 清除覆蓋語意（見 docs/decisions/driver-report-import-overwrite.md）。獨立成一張表，
-- 不在 ride_sources 加狀態欄位，避免動到已完整測試的混車合併查詢路徑。
CREATE TABLE ride_source_row_conflicts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    form_id UUID NOT NULL REFERENCES driver_report_forms(id) ON DELETE CASCADE,
    vehicle_id UUID NOT NULL REFERENCES vehicles(id) ON DELETE RESTRICT,
    case_id UUID NOT NULL REFERENCES cases(id) ON DELETE RESTRICT,
    service_date DATE NOT NULL,
    leg_seq SMALLINT NOT NULL CHECK (leg_seq BETWEEN 1 AND 4),
    source_column_index INT NOT NULL,
    previous_submission_id UUID REFERENCES form_submissions(id) ON DELETE SET NULL,
    previous_reported TEXT NOT NULL CHECK (previous_reported IN ('boarded', 'absent')),
    previous_driver_id UUID REFERENCES drivers(id) ON DELETE SET NULL,
    previous_submitted_at TIMESTAMPTZ NOT NULL,
    new_submission_id UUID NOT NULL REFERENCES form_submissions(id) ON DELETE CASCADE,
    new_reported TEXT NOT NULL CHECK (new_reported IN ('boarded', 'absent')),
    new_driver_id UUID REFERENCES drivers(id) ON DELETE SET NULL,
    new_submitted_at TIMESTAMPTZ NOT NULL,
    detected_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at TIMESTAMPTZ,
    resolved_by UUID,
    resolution TEXT CHECK (resolution IN ('kept_previous', 'used_new'))
);

-- 同一 slot 同時只允許一筆未解決的衝突：第二次上傳到同一個未解決的衝突直接更新其
-- 新值，previous_* 保持不動——這是「待維護若是同一筆以最新上傳為準」的資料庫層保證。
CREATE UNIQUE INDEX uq_ride_source_row_conflict_open
    ON ride_source_row_conflicts (vehicle_id, case_id, service_date, leg_seq)
    WHERE resolved_at IS NULL;

-- 供待維護清單依表單彙整未解決的衝突。
CREATE INDEX ix_ride_source_row_conflicts_form_open
    ON ride_source_row_conflicts (form_id)
    WHERE resolved_at IS NULL;
