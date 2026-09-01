-- 政府申報匯出歷史紀錄需顯示操作人員，JWT 的 user_metadata.display_name 只在當下有效，
-- 建立當下落地成快照欄位，帳號日後被刪除或改名也不影響歷史紀錄的可讀性。
ALTER TABLE export_jobs ADD COLUMN created_by_name TEXT NOT NULL DEFAULT '';
