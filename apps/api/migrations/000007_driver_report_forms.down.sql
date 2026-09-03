-- 還原為 Google 表單同步的結構。已被 DROP 的 sheet_id／ingest_secret_ref 值無法復原，
-- 只能以空字串重建欄位。
ALTER TABLE form_submissions ALTER COLUMN source SET DEFAULT 'webhook';
ALTER TABLE form_submissions DROP CONSTRAINT IF EXISTS form_submissions_source_check;
ALTER TABLE form_submissions ADD CONSTRAINT form_submissions_source_check CHECK (source IN ('webhook', 'sheets_sync', 'manual'));
UPDATE form_submissions SET source = 'webhook' WHERE source = 'import';

ALTER TABLE form_columns DROP CONSTRAINT IF EXISTS uq_form_column_header;
ALTER TABLE form_columns ADD CONSTRAINT uq_form_column_idx UNIQUE (form_id, column_index);

DROP INDEX IF EXISTS uq_driver_report_forms_vehicle;

ALTER TABLE driver_report_forms RENAME COLUMN last_imported_at TO last_synced_at;
ALTER TABLE driver_report_forms ADD COLUMN ingest_secret_ref TEXT NOT NULL DEFAULT '';
ALTER TABLE driver_report_forms ADD COLUMN sheet_id TEXT NOT NULL DEFAULT '';
ALTER TABLE driver_report_forms RENAME TO google_forms;
CREATE UNIQUE INDEX IF NOT EXISTS google_forms_sheet_id_key ON google_forms(sheet_id);
