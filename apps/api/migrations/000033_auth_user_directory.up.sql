-- Migration: 000033_auth_user_directory.up.sql
-- Description: 建立 Supabase Auth 使用者的本地清單查詢投影。

CREATE TABLE auth_user_directory (
    user_id UUID PRIMARY KEY,
    email TEXT NOT NULL,
    display_name TEXT NOT NULL DEFAULT '',
    phone TEXT NOT NULL DEFAULT '',
    role_key TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    custom_permissions JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    last_sign_in_at TIMESTAMPTZ
);

CREATE INDEX idx_auth_user_directory_role_key
    ON auth_user_directory (role_key);

CREATE INDEX idx_auth_user_directory_email
    ON auth_user_directory (email);

ALTER TABLE auth_user_directory ENABLE ROW LEVEL SECURITY;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'anon') THEN
        REVOKE ALL ON TABLE public.auth_user_directory FROM anon;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'authenticated') THEN
        REVOKE ALL ON TABLE public.auth_user_directory FROM authenticated;
    END IF;
END $$;
