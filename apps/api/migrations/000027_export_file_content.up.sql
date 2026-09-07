ALTER TABLE export_job_files
    ADD COLUMN storage_path TEXT,
    ADD COLUMN file_content BYTEA,
    ADD COLUMN file_size BIGINT;

COMMENT ON COLUMN export_job_files.file_content IS
    '匯出成功當下的完整 XLSX 位元組；歷史下載不得依目前主檔重建';
