-- Migration: 000011_backfill_admin_identity.down.sql
-- Description: 回滾補登的預設管理員 auth.identities。

DO $$
BEGIN
IF EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'auth') THEN
    DELETE FROM auth.identities WHERE user_id = '00000000-0000-0000-0000-000000000002' AND provider = 'email';
END IF;
END $$;
