-- Security hardening is intentionally irreversible through the generic down command.
-- Refuse the rollback before the migration runner can delete schema_migrations, so a failed
-- operator attempt cannot make the database look rolled back while leaving hidden hardening.
DO $$
BEGIN
    RAISE EXCEPTION 'migration 000054 is irreversible security hardening; rollback is refused to keep public table lockdown enabled';
END;
$$;
