-- Migration: 000053_form_columns_mapping_integrity.down.sql
-- Description: 移除 form_columns 的 mapped/case_id 一致性約束與個案刪除降級觸發。
--
-- 注意：up 的第 1 步（把孤兒列降級成 pending）不可逆，這裡也不嘗試還原。
-- 那些列原本的 case_id 在外鍵 SET NULL 當下就已經消失，資料庫裡沒有任何資訊
-- 可以區分「本來就是 pending」與「被這支 migration 降級成 pending」。
-- 比照 000043_region_master_fk.down.sql 的既有慣例，只還原結構、不還原資料。

ALTER TABLE form_columns DROP CONSTRAINT IF EXISTS ck_form_columns_mapped_ride_requires_case;

DROP TRIGGER IF EXISTS trg_cases_degrade_form_columns ON cases;
DROP FUNCTION IF EXISTS degrade_form_columns_on_case_delete();
