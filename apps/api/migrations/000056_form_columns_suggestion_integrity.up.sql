-- Migration: 000056_form_columns_suggestion_integrity.up.sql
-- Description: 修復並防堵「suggested_case_id 已清空但 suggestion_score 未歸零」的匯報表欄位推薦孤兒列。
--
-- 成因：suggested_case_id 的外鍵是 ON DELETE SET NULL，個案被刪除／合併時會把
-- suggested_case_id 清成 NULL，但 suggestion_score 沒有任何連動機制一起歸零；UpsertColumns
-- 舊版 upsert 又用 COALESCE(既有值, 新值) 保留舊 id、GREATEST(既有值, 新值) 保留歷史最高分，
-- 兩個欄位各自從不同來源合併，score 只會愈疊愈高、不會因對應個案消失而下修。結果就是待維護
-- 清單顯示「系統推薦：無相符個案（信心度 100%）」這種語意矛盾的畫面，且下拉選單能選到的個案
-- 跟 suggested_case_id 早已失聯的舊建議完全對不上。

-- 1. 修復既有資料：id 已是 NULL 的推薦分數歸零。
UPDATE form_columns
SET suggestion_score = 0,
    updated_at = now()
WHERE suggested_case_id IS NULL
  AND suggestion_score > 0;

-- 2. 修行為：個案被硬刪除時，連同 suggestion_score 一起歸零，不留下孤兒分數。
--    比照 000053_form_columns_mapping_integrity 對 case_id/mapping_status 的作法，用
--    BEFORE DELETE row trigger 早於外鍵的 referential action 執行。
CREATE OR REPLACE FUNCTION degrade_form_columns_suggestion_on_case_delete() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    UPDATE form_columns
    SET suggested_case_id = NULL,
        suggestion_score = 0,
        updated_at = now()
    WHERE suggested_case_id = OLD.id;
    RETURN OLD;
END;
$$;

DROP TRIGGER IF EXISTS trg_cases_degrade_form_columns_suggestion ON cases;
CREATE TRIGGER trg_cases_degrade_form_columns_suggestion
BEFORE DELETE ON cases
FOR EACH ROW
EXECUTE FUNCTION degrade_form_columns_suggestion_on_case_delete();

-- 3. 最後一道防線：suggestion_score 不該在沒有 suggested_case_id 時維持正值。
--    應用層（driver_report_repo.go 的 UpsertColumns）已改成兩欄成對覆蓋，這條 CHECK
--    防的是直接下 SQL 與未來新增的寫入路徑。
ALTER TABLE form_columns
ADD CONSTRAINT ck_form_columns_suggestion_score_requires_case
CHECK (suggested_case_id IS NOT NULL OR suggestion_score = 0);
