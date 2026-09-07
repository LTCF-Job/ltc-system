-- 只還原 schema 約束；已回填的空字串不還原為 NULL（回填內容本身不是這次要 revert 的資料錯誤）。
ALTER TABLE ride_records ALTER COLUMN based_on_fingerprint DROP NOT NULL;
ALTER TABLE ride_records ALTER COLUMN based_on_fingerprint DROP DEFAULT;
