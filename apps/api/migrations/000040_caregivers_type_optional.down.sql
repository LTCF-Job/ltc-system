-- 還原約束前必須清掉空字串類型，否則約束無法重建。這些列全部來自匯入且尚未補齊類型，
-- 回填任一類型都是捏造資料，因此直接刪除（與 000015、000020 down 的破壞性還原慣例一致）。

DELETE FROM caregivers WHERE type = '';

ALTER TABLE caregivers DROP CONSTRAINT IF EXISTS caregivers_type_check;
ALTER TABLE caregivers ADD CONSTRAINT caregivers_type_check CHECK (type IN ('case_manager', 'specialist'));
