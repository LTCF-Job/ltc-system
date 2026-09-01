-- Migration: 000011_backfill_admin_identity.down.sql
-- Description: 回滾補登的預設管理員 auth.identities。

DELETE FROM auth.identities WHERE user_id = '00000000-0000-0000-0000-000000000002' AND provider = 'email';
