-- Migration: 000056_form_columns_suggestion_integrity.down.sql
-- Description: 移除 form_columns 的 suggestion_score/suggested_case_id 一致性約束與降級觸發。
--
-- 注意：up 的第 1 步（把孤兒推薦分數歸零）不可逆，這裡也不嘗試還原。
-- 比照 000053_form_columns_mapping_integrity.down.sql 的既有慣例，只還原結構、不還原資料。

ALTER TABLE form_columns DROP CONSTRAINT IF EXISTS ck_form_columns_suggestion_score_requires_case;

DROP TRIGGER IF EXISTS trg_cases_degrade_form_columns_suggestion ON cases;
DROP FUNCTION IF EXISTS degrade_form_columns_suggestion_on_case_delete();
