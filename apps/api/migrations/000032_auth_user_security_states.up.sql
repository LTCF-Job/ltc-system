-- Migration: 000032_auth_user_security_states.up.sql
-- Description: 建立跨 API replica 共用的帳號停用與個人權限投影版本。

CREATE TABLE auth_user_security_states (
    user_id UUID PRIMARY KEY,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    role_key TEXT NOT NULL DEFAULT '',
    custom_permissions JSONB NOT NULL DEFAULT '{}'::jsonb,
    permission_version BIGINT NOT NULL DEFAULT 1,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_auth_user_security_states_status
    ON auth_user_security_states (status);
