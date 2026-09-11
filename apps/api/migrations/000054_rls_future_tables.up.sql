-- Migration: 000054_rls_future_tables.up.sql
-- Description: 將 000028 之後新增的資料表納入同一套 RLS 與預設權限封鎖。
--
-- 000028 只鎖定當時已存在的 public tables；後續新增的表若沒有再次套用
-- ENABLE ROW LEVEL SECURITY，Supabase 的 anon/authenticated role 可能直接讀寫。
-- 這裡以目前 migration connection 的 owner 設定 default privileges，讓之後由同一
-- owner 建立的 public table／sequence 也會維持封鎖；不同 owner 則由下方 event trigger
-- 補上相同的 table／sequence 防護。
-- CREATE EVENT TRIGGER 僅允許 superuser；migration identity 若不具備該權限會在同一
-- transaction 內失敗，避免只完成一半的安全設定。

DO $$
DECLARE
    table_record record;
BEGIN
    FOR table_record IN
        SELECT schemaname, tablename
        FROM pg_catalog.pg_tables
        WHERE schemaname = 'public'
    LOOP
        EXECUTE format(
            'ALTER TABLE %I.%I ENABLE ROW LEVEL SECURITY',
            table_record.schemaname,
            table_record.tablename
        );

        IF EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = 'anon') THEN
            EXECUTE format(
                'REVOKE ALL ON TABLE %I.%I FROM anon',
                table_record.schemaname,
                table_record.tablename
            );
        END IF;
        IF EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = 'authenticated') THEN
            EXECUTE format(
                'REVOKE ALL ON TABLE %I.%I FROM authenticated',
                table_record.schemaname,
                table_record.tablename
            );
        END IF;
    END LOOP;

    IF EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = 'anon') THEN
        REVOKE ALL ON ALL TABLES IN SCHEMA public FROM anon;
        REVOKE ALL ON ALL SEQUENCES IN SCHEMA public FROM anon;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = 'authenticated') THEN
        REVOKE ALL ON ALL TABLES IN SCHEMA public FROM authenticated;
        REVOKE ALL ON ALL SEQUENCES IN SCHEMA public FROM authenticated;
    END IF;

    -- PUBLIC 權限會被 anon／authenticated 繼承，不能只撤銷兩個具名角色。
    REVOKE ALL ON ALL TABLES IN SCHEMA public FROM PUBLIC;
    REVOKE ALL ON ALL SEQUENCES IN SCHEMA public FROM PUBLIC;
END;
$$;

-- ALTER DEFAULT PRIVILEGES 只影響執行這段 migration 的 database owner；這是刻意的，
-- 避免把不屬於本專案的其他 owner 權限改掉。多 owner 部署仍由下方 event trigger
-- 補上 runtime 防護；若要移除 event trigger，必須先確認所有 owner 的 default privileges。
DO $$
BEGIN
    ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE ALL ON TABLES FROM PUBLIC;
    ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE ALL ON SEQUENCES FROM PUBLIC;
    IF EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = 'anon') THEN
        ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE ALL ON TABLES FROM anon;
        ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE ALL ON SEQUENCES FROM anon;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = 'authenticated') THEN
        ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE ALL ON TABLES FROM authenticated;
        ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE ALL ON SEQUENCES FROM authenticated;
    END IF;
END;
$$;

-- Default privileges 只能控制 grant，不能讓未來新 table 自動開啟 RLS，且不同 owner
-- 可能有自己的 default privileges。event trigger 以 DDL 結束事件補上兩層防護，確保
-- 任何 owner 在 public 建立的 table／sequence 都不會把 Data API 權限重新打開。
CREATE OR REPLACE FUNCTION public.enforce_public_table_lockdown()
RETURNS event_trigger
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog
AS $$
DECLARE
    object_record record;
    previous_guard text := current_setting('ltc.public_table_lockdown_active', true);
BEGIN
    -- 函式內的 ALTER／REVOKE 也會觸發本規則；用 transaction-local GUC
    -- 避免遞迴，同時在外層函式結束時還原狀態，避免影響同一 transaction 的後續 DDL。
    IF previous_guard = 'on' THEN
        RETURN;
    END IF;
    PERFORM set_config('ltc.public_table_lockdown_active', 'on', true);

    -- GRANT／REVOKE 的 event-trigger metadata 沒有提供受影響 object 的 OID；
    -- 這兩類事件改為重掃 public objects，確保直接授權也會被撤回。
    IF tg_tag IN ('GRANT', 'REVOKE') THEN
        FOR object_record IN
            SELECT c.oid, c.relkind
            FROM pg_class c
            JOIN pg_namespace n ON n.oid = c.relnamespace
            WHERE n.nspname = 'public'
              AND c.relkind IN ('r', 'p', 'S')
        LOOP
            IF object_record.relkind IN ('r', 'p') THEN
                EXECUTE format('ALTER TABLE %s ENABLE ROW LEVEL SECURITY', object_record.oid::regclass);
                EXECUTE format('REVOKE ALL ON TABLE %s FROM PUBLIC', object_record.oid::regclass);
                IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'anon') THEN
                    EXECUTE format('REVOKE ALL ON TABLE %s FROM anon', object_record.oid::regclass);
                END IF;
                IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'authenticated') THEN
                    EXECUTE format('REVOKE ALL ON TABLE %s FROM authenticated', object_record.oid::regclass);
                END IF;
            ELSE
                EXECUTE format('REVOKE ALL ON SEQUENCE %s FROM PUBLIC', object_record.oid::regclass);
                IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'anon') THEN
                    EXECUTE format('REVOKE ALL ON SEQUENCE %s FROM anon', object_record.oid::regclass);
                END IF;
                IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'authenticated') THEN
                    EXECUTE format('REVOKE ALL ON SEQUENCE %s FROM authenticated', object_record.oid::regclass);
                END IF;
            END IF;
        END LOOP;

        PERFORM set_config('ltc.public_table_lockdown_active', coalesce(previous_guard, 'off'), true);
        RETURN;
    END IF;

    FOR object_record IN
        SELECT objid, object_type
        FROM pg_event_trigger_ddl_commands()
        WHERE schema_name = 'public'
          AND object_type IN ('table', 'sequence')
    LOOP
        IF object_record.object_type = 'table' THEN
            EXECUTE format('ALTER TABLE %s ENABLE ROW LEVEL SECURITY', object_record.objid::regclass);
            EXECUTE format('REVOKE ALL ON TABLE %s FROM PUBLIC', object_record.objid::regclass);
            IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'anon') THEN
                EXECUTE format('REVOKE ALL ON TABLE %s FROM anon', object_record.objid::regclass);
            END IF;
            IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'authenticated') THEN
                EXECUTE format('REVOKE ALL ON TABLE %s FROM authenticated', object_record.objid::regclass);
            END IF;
        ELSE
            EXECUTE format('REVOKE ALL ON SEQUENCE %s FROM PUBLIC', object_record.objid::regclass);
            IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'anon') THEN
                EXECUTE format('REVOKE ALL ON SEQUENCE %s FROM anon', object_record.objid::regclass);
            END IF;
            IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'authenticated') THEN
                EXECUTE format('REVOKE ALL ON SEQUENCE %s FROM authenticated', object_record.objid::regclass);
            END IF;
        END IF;
    END LOOP;

    PERFORM set_config('ltc.public_table_lockdown_active', coalesce(previous_guard, 'off'), true);
END;
$$;

REVOKE ALL ON FUNCTION public.enforce_public_table_lockdown() FROM PUBLIC;

DROP EVENT TRIGGER IF EXISTS enforce_public_table_lockdown;
CREATE EVENT TRIGGER enforce_public_table_lockdown
    ON ddl_command_end
    WHEN TAG IN ('CREATE TABLE', 'CREATE TABLE AS', 'SELECT INTO', 'CREATE SEQUENCE', 'ALTER TABLE', 'ALTER SEQUENCE', 'GRANT', 'REVOKE')
    EXECUTE FUNCTION public.enforce_public_table_lockdown();
