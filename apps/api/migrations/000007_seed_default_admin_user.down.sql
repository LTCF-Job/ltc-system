-- Migration: 000007_seed_default_admin_user.down.sql
-- Description: 移除預設 Supabase Auth 管理員帳號

DELETE FROM auth.users
WHERE id = '00000000-0000-0000-0000-000000000002';
