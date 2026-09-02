-- Migration: 000011_backfill_admin_identity.up.sql
-- Description: 補回 000002 遺漏的預設管理員 auth.identities（provider = 'email'）。
-- 已跑過舊版 000002 的環境（auth.users 已存在但缺 identities）需要這支補上，
-- 否則密碼正確仍會被 Supabase Auth 判定 Invalid login credentials。

-- ltc_demo 等沒有 auth schema 的資料庫整段略過。
DO $$
BEGIN
IF EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'auth') THEN

INSERT INTO auth.identities (
    id,
    provider_id,
    user_id,
    identity_data,
    provider,
    last_sign_in_at,
    created_at,
    updated_at
)
SELECT
    '00000000-0000-0000-0000-000000000002',
    '00000000-0000-0000-0000-000000000002',
    '00000000-0000-0000-0000-000000000002',
    '{"sub":"00000000-0000-0000-0000-000000000002","email":"ltcf-admin@ltc.example.com","email_verified":true,"phone_verified":false}'::jsonb,
    'email',
    now(),
    now(),
    now()
WHERE EXISTS (
    SELECT 1 FROM auth.users WHERE id = '00000000-0000-0000-0000-000000000002'
)
ON CONFLICT (provider_id, provider) DO UPDATE SET
    identity_data = EXCLUDED.identity_data,
    updated_at = now();

END IF;
END $$;
