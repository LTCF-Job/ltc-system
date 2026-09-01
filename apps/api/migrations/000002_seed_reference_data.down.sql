-- Migration: 000002_seed_reference_data.down.sql
-- Description: 回滾正式參考資料與預設管理員帳號

DELETE FROM auth.identities WHERE user_id = '00000000-0000-0000-0000-000000000002';
DELETE FROM auth.users WHERE id = '00000000-0000-0000-0000-000000000002';
DELETE FROM regions;
