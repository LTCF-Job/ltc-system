-- 司機接送匯報：資料來源由 Google 試算表同步改為 .xlsx 檔案匯入。
-- google_forms 更名為 driver_report_forms，並移除 Google 專屬欄位。

ALTER TABLE google_forms RENAME TO driver_report_forms;

DROP INDEX IF EXISTS google_forms_sheet_id_key;
ALTER TABLE driver_report_forms DROP COLUMN IF EXISTS sheet_id;
ALTER TABLE driver_report_forms DROP COLUMN IF EXISTS ingest_secret_ref;
ALTER TABLE driver_report_forms RENAME COLUMN last_synced_at TO last_imported_at;

-- 同一台車的匯報表只需一份，避免重複建立造成欄位對應分裂。
CREATE UNIQUE INDEX IF NOT EXISTS uq_driver_report_forms_vehicle ON driver_report_forms(vehicle_id);

-- 欄位對應改以表頭文字為識別。個案增減會讓同一個案在不同月份落在不同欄號，
-- 以欄號當唯一鍵會在下次匯入時錯配到別的個案。
ALTER TABLE form_columns DROP CONSTRAINT IF EXISTS uq_form_column_idx;
ALTER TABLE form_columns ADD CONSTRAINT uq_form_column_header UNIQUE (form_id, column_header);

-- 匯報來源只剩「檔案匯入」與「人工補登」兩種。
UPDATE form_submissions SET source = 'import' WHERE source IN ('webhook', 'sheets_sync');
ALTER TABLE form_submissions DROP CONSTRAINT IF EXISTS form_submissions_source_check;
ALTER TABLE form_submissions ADD CONSTRAINT form_submissions_source_check CHECK (source IN ('import', 'manual'));
ALTER TABLE form_submissions ALTER COLUMN source SET DEFAULT 'import';
