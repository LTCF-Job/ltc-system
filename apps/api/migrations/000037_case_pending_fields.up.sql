-- 個案批次匯入待維護擴充：生日/身分證字號格式錯誤比照單位/車輛既有的
-- raw 欄位模式，不擋列匯入，改為推導式待維護，待維護頁就地補正。

ALTER TABLE cases ADD COLUMN birth_date_raw TEXT;
ALTER TABLE cases ADD COLUMN national_id_invalid BOOLEAN NOT NULL DEFAULT false;
