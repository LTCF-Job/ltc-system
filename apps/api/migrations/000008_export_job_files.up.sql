-- 政府申報匯出改為「一個個案一個月一份檔案」。原本的 export_jobs.storage_path
-- 只能記一個檔案路徑，撐不住一次匯出多份工作簿，因此把逐案檔案獨立成一張表。

CREATE TABLE export_job_files (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_id UUID NOT NULL REFERENCES export_jobs(id) ON DELETE CASCADE,
    case_id UUID NOT NULL REFERENCES cases(id) ON DELETE RESTRICT,
    seq INT NOT NULL,
    -- 個案編號、姓名與區域是匯出當下的快照；個案之後改名或轉區時歷史紀錄仍呈現當時的內容
    case_code TEXT NOT NULL,
    case_name TEXT NOT NULL,
    region TEXT NOT NULL,
    file_name TEXT NOT NULL,
    row_count INT NOT NULL,
    file_checksum TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_export_job_file_case UNIQUE (job_id, case_id),
    CONSTRAINT uq_export_job_file_seq UNIQUE (job_id, seq)
);

-- 下載時以 (job_id, case_id) 撈回該案的申報列並依 line_no 重繪工作簿
CREATE INDEX idx_export_lines_job_case ON export_lines(job_id, case_id, line_no);
