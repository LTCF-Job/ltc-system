DO $$
DECLARE
    item record;
BEGIN
    FOR item IN
        SELECT tablename FROM pg_catalog.pg_tables WHERE schemaname = 'public'
    LOOP
        EXECUTE format('ALTER TABLE public.%I DISABLE ROW LEVEL SECURITY', item.tablename);
    END LOOP;
END $$;
