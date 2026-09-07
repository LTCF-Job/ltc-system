-- 只還原欄位與唯一鍵設定；被上層 up migration 刪除的重複歷史紀錄無法復原。
ALTER TABLE ride_sources DROP COLUMN IF EXISTS submitted_at;

ALTER TABLE form_submissions DROP CONSTRAINT IF EXISTS uq_form_submission;
ALTER TABLE form_submissions ADD CONSTRAINT uq_form_submission UNIQUE (form_id, service_date, submitted_at);
