-- 還原後 code 一律為 NULL，需人工回補才能重新套用 NOT NULL UNIQUE。
ALTER TABLE sites ADD COLUMN code TEXT UNIQUE;
