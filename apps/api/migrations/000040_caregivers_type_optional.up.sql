-- Migration: 000040_caregivers_type_optional.up.sql
-- Description: 批次匯入改為「缺欄位只留白、不擋列」，類型缺漏或不是個管／專護時以空字串
-- 寫入並列入待維護補齊，因此放寬 type 的 CHECK 讓空字串合法。手動建立與編輯仍在應用層
-- 要求二選一，空字串只會由匯入產生。

ALTER TABLE caregivers DROP CONSTRAINT IF EXISTS caregivers_type_check;
ALTER TABLE caregivers ADD CONSTRAINT caregivers_type_check CHECK (type IN ('case_manager', 'specialist', ''));
