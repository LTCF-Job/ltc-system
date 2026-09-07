-- 匯報改成逐列比對後不再整段清除覆蓋（見 docs/decisions/driver-report-import-overwrite.md）。
-- 舊唯一鍵 (form_id, service_date, submitted_at) 在跨次上傳時形同虛設：submitted_at
-- 每次上傳都是新時間戳，這個鍵永遠不會撞號，一車一天一筆全靠額外的「先刪後寫」保證。
-- 現在改成資料庫層級真正保證一車一天只有一筆現行 form_submissions，讓同一天的重複
-- 上傳原地更新（ON CONFLICT (form_id, service_date)），而不是疊加出多筆。

-- 清理既有重複：同一台車同一天若過去曾被重複上傳、留下多筆歷史紀錄，只保留
-- submitted_at 最新的一筆（同分秒時以 id 穩定排序決勝），其餘刪除；ride_sources 由
-- ON DELETE CASCADE 連帶清除——這些本來就是被混車合併演算法忽略、對外沒有實際生效
-- 的舊資料。此步驟不可逆，正式環境執行前需先備份資料庫。
DELETE FROM form_submissions fs
WHERE fs.id IN (
    SELECT id FROM (
        SELECT id,
               row_number() OVER (
                   PARTITION BY form_id, service_date
                   ORDER BY submitted_at DESC, id DESC
               ) AS rn
        FROM form_submissions
    ) ranked
    WHERE ranked.rn > 1
);

ALTER TABLE form_submissions DROP CONSTRAINT IF EXISTS uq_form_submission;
ALTER TABLE form_submissions ADD CONSTRAINT uq_form_submission UNIQUE (form_id, service_date);

-- form_submissions.submitted_at 從此會在同一天重傳時原地更新，不再是這一列曾經寫入
-- 當下的時間戳。ride_sources 的混車合併排序（sourcePreferred）原本借用它判斷「這台車
-- 這筆來源是什麼時候寫入的」，繼續借用會讓同一天較早寫入、本次未變動的搭乘來源，
-- 因為同一天稍後的另一次上傳而被誤判成剛剛才寫入，進而在跨車衝突判斷中贏過真正
-- 較新的來源。改成每筆 ride_sources 各自記錄自己寫入當下的時間，不再依賴聯表。
ALTER TABLE ride_sources ADD COLUMN submitted_at TIMESTAMPTZ;
UPDATE ride_sources rs
SET submitted_at = fs.submitted_at
FROM form_submissions fs
WHERE fs.id = rs.submission_id AND rs.submitted_at IS NULL;
ALTER TABLE ride_sources ALTER COLUMN submitted_at SET NOT NULL;
