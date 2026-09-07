-- Migration: 000041_pending_flag_columns.down.sql
-- Description: 還原待維護判定欄位與 view。generated column 由來源欄位推導，刪除不會遺失任何
-- 使用者資料；view 先於欄位刪除，避免相依性阻擋。

DROP VIEW IF EXISTS case_pending_status;

DROP INDEX IF EXISTS ix_caregivers_is_pending;
DROP INDEX IF EXISTS ix_case_transport_preferences_link_pending;
DROP INDEX IF EXISTS ix_cases_profile_pending;

ALTER TABLE caregivers DROP COLUMN IF EXISTS is_pending;
ALTER TABLE case_transport_preferences DROP COLUMN IF EXISTS link_pending;
ALTER TABLE cases DROP COLUMN IF EXISTS profile_pending;
