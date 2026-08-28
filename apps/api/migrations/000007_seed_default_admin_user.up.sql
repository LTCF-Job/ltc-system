-- Migration: 000007_seed_default_admin_user.up.sql
-- Description: 建立預設 Supabase Auth 管理員帳號

INSERT INTO auth.users (
    id,
    instance_id,
    aud,
    role,
    email,
    encrypted_password,
    email_confirmed_at,
    raw_app_meta_data,
    raw_user_meta_data,
    created_at,
    updated_at,
    confirmation_token,
    recovery_token,
    email_change_token_new,
    email_change
)
VALUES (
    '00000000-0000-0000-0000-000000000002',
    '00000000-0000-0000-0000-000000000000',
    'authenticated',
    'authenticated',
    'ltcf-admin@ltc.example.com',
    '$2a$10$IKiM.bIaxDYbD/.cl.b8MODthsuQ0WhLDcvox90gC3H3TDHaHFVYe',
    now(),
    '{"provider":"email","providers":["email"],"role":"admin"}'::jsonb,
    '{"display_name":"系統管理員"}'::jsonb,
    now(),
    now(),
    '', '', '', ''
)
ON CONFLICT (id) DO UPDATE SET
    email = EXCLUDED.email,
    encrypted_password = EXCLUDED.encrypted_password,
    email_confirmed_at = EXCLUDED.email_confirmed_at,
    raw_app_meta_data = EXCLUDED.raw_app_meta_data,
    raw_user_meta_data = EXCLUDED.raw_user_meta_data,
    updated_at = now();
