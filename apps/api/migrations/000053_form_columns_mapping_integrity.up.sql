-- Migration: 000053_form_columns_mapping_integrity.up.sql
-- Description: 修復並防堵「已對應卻沒有個案」的匯報表欄位對應孤兒列。
--
-- 成因：form_columns.case_id 的外鍵是 ON DELETE SET NULL。個案主檔被硬刪除重建時
-- （例如重新匯入個案、直接下 SQL 刪除），case_id 被清成 NULL，但 mapping_status 仍留在
-- 'mapped'。這個組合有兩個後果，而且兩個都沒有任何畫面看得到：
--   1. ride 展開邏輯（RideService.IngestWebhook）會跳過 case_id 為空的欄位，該欄從此
--      展不出任何 ride_source／ride_record，匯報表上傳了也等於沒上傳。
--   2. 待維護頁的 ListColumnsWithMapping 依 mapping_status 篩選，'mapped' 不會出現在
--      待維護清單，使用者連「這一欄需要重新對應」都看不到。
-- 結果就是匯報資料靜默消失，一路到政府申報匯出才發現「沒有任何資料」。

-- 1. 修復既有資料：只降級 kind = 'ride'。
--    meta（時間戳記、今日駕駛人）與 issue（問題回報）欄位本來就不對應任何個案，
--    它們的 mapped + case_id IS NULL 是正確狀態，不能一起降級，3. 的 CHECK 也必須放行。
UPDATE form_columns
SET mapping_status = 'pending',
    leg_seq = NULL,
    updated_at = now()
WHERE mapping_status = 'mapped'
  AND kind = 'ride'
  AND case_id IS NULL;

-- 2. 修行為：個案被硬刪除時，把對應降級成待維護，而不是讓外鍵把 case_id 抹成 NULL
--    卻留著 mapped。
--
--    這裡用 BEFORE DELETE row trigger 而不是改外鍵動作：
--      - ON DELETE SET NULL 正是造成孤兒的原因，而且會直接違反 3. 的 CHECK，刪除會失敗。
--      - ON DELETE CASCADE 會連整列欄位定義一起刪掉，連帶失去 column_header 既有的
--        推薦值與歷史，使用者下次匯入才會重新看到這一欄。
--    BEFORE DELETE row trigger 早於外鍵的 referential action 執行，因此先降級再讓外鍵
--    去更新已經是 NULL 的 case_id，兩者不會互相打架。
CREATE OR REPLACE FUNCTION degrade_form_columns_on_case_delete() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    UPDATE form_columns
    SET mapping_status = 'pending',
        case_id = NULL,
        leg_seq = NULL,
        updated_at = now()
    WHERE case_id = OLD.id
      AND mapping_status = 'mapped';
    RETURN OLD;
END;
$$;

DROP TRIGGER IF EXISTS trg_cases_degrade_form_columns ON cases;
CREATE TRIGGER trg_cases_degrade_form_columns
BEFORE DELETE ON cases
FOR EACH ROW
EXECUTE FUNCTION degrade_form_columns_on_case_delete();

-- 3. 最後一道防線。應用層（commit.go 的 persistColumnDecisions 與
--    DriverReportService.UpdateColumnMapping）已經擋住這個組合，這條 CHECK 防的是
--    直接下 SQL 與未來新增的寫入路徑。
ALTER TABLE form_columns
ADD CONSTRAINT ck_form_columns_mapped_ride_requires_case
CHECK (mapping_status <> 'mapped' OR kind <> 'ride' OR case_id IS NOT NULL);
