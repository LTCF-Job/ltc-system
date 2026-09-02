-- demo-db-roles.sql
--
-- 用途：讓 Demo 資料庫（ltc_demo）與正式資料庫（postgres）在 Postgres 角色層級互相隔離，
-- 避免 Demo API 的連線角色讀寫到正式資料，或正式 API 的連線角色連到 ltc_demo。
--
-- 這是一次性的 cluster-admin 操作，不是每次部署都要跑的 schema migration，因此刻意不放進
-- apps/api/cmd/migrate（該工具會把同一份檔案內容套用到 production 與 ltc_demo 兩個資料庫，
-- 前提是兩邊套用結果要一致；本檔案的 GRANT／REVOKE 卻是刻意兩邊不對稱，硬塞進 migrate
-- 反而會需要在檔案內判斷「目前連的是哪個資料庫」，徒增複雜度）。
--
-- 執行方式：由有管理員權限的人，透過 Supabase SQL Editor 或 psql，「連線到 ltc_demo 資料庫」
-- 執行整份檔案一次：
--
--     psql "<ltc_demo 的連線字串>" -f apps/api/ops/demo-db-roles.sql
--
-- 之所以只能連 ltc_demo 執行：Postgres 的角色（ROLE）是整個 cluster 共用的，但 GRANT 授予
-- 的資料表權限是「當下連線的資料庫」專屬的；REVOKE CONNECT 則是唯一例外，對任何一個資料庫
-- 下達都會立即生效，不需要連到目標資料庫本身。整份腳本因此假設是在連到 ltc_demo 時執行。
--
-- 重複執行安全（idempotent）：CREATE ROLE 用 DO 區塊搭配存在性檢查模擬
-- "CREATE ROLE IF NOT EXISTS"（Postgres 沒有原生語法），GRANT／REVOKE 本身即可重複執行。

-- 1. 建立 Demo API 專用的連線角色（若尚未存在）。
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'ltc_demo_app') THEN
        -- CHANGE ME：這裡的密碼只是佔位字串，正式建立後務必立刻在 Supabase Dashboard
        -- 或用 ALTER ROLE ltc_demo_app WITH PASSWORD '<新密碼>' 換成真正的密碼，
        -- 再把新密碼寫進 Demo Cloud Run 服務的 DATABASE_URL 密鑰。
        CREATE ROLE ltc_demo_app WITH LOGIN PASSWORD 'CHANGE_ME_ROTATE_BEFORE_USE';
    END IF;
END
$$;

-- 2. 正式環境角色：若 DATABASE_URL 目前用的是 Supabase 專案預設的 postgres 超級使用者，
--    此腳本不會另外建立 ltc_prod_app 去取代它——貿然切換連線角色屬於正式環境變更，
--    應該另外排時間驗證後再做，不適合夾在「建 Demo 資料庫」這個操作裡一併處理。
--    這裡只建立這個角色供之後遷移使用；正式 DATABASE_URL 要不要換成這個角色是後續追蹤事項，
--    在下方 REVOKE 前先確認清楚現在的正式連線用的是哪個角色，避免誤鎖自己。
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'ltc_prod_app') THEN
        -- CHANGE ME：同上，正式使用前先在 Supabase Dashboard 或用 ALTER ROLE 換成真正密碼。
        CREATE ROLE ltc_prod_app WITH LOGIN PASSWORD 'CHANGE_ME_ROTATE_BEFORE_USE';
    END IF;
END
$$;

-- 3. 授予 ltc_demo_app 在 ltc_demo（目前連線的資料庫）裡，對 public schema 全部既有資料表
--    的 DML 權限，以及序列（SERIAL/IDENTITY 欄位的自動遞增）所需的 USAGE 權限。
GRANT USAGE ON SCHEMA public TO ltc_demo_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO ltc_demo_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO ltc_demo_app;

-- 4. 未來 migrations 新增的資料表／序列，只要是由目前執行本腳本的角色建立，也會自動套用
--    同一組權限，不必每次新增表格後重跑本腳本。
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO ltc_demo_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT USAGE, SELECT ON SEQUENCES TO ltc_demo_app;

-- 5. 實際隔離機制：Postgres 的 CONNECT 權限判斷是「聯集制」——只要角色本身、其所屬群組、
--    或 PUBLIC 三者之中任何一個有 GRANT，就能連線；對單一角色下 REVOKE CONNECT，只要
--    PUBLIC 仍保有 CONNECT，該角色照樣連得進去（此腳本開發時已用本機 Postgres 16 容器
--    實際驗證過這個行為，見 PR/commit 說明）。因此要讓 REVOKE 真的生效，必須先收回
--    PUBLIC 的 CONNECT，再把合法需要連線的角色個別加回來。
--
--    ltc_demo 是這個 Demo 功能新建、專屬給 Demo API 使用的資料庫，沒有其他既有角色依賴它，
--    收回 PUBLIC CONNECT 後只重新授權給 ltc_demo_app 是安全的，這一步能確實擋下
--    ltc_prod_app（以及其他任何未被列出的角色）連進 ltc_demo。
REVOKE CONNECT ON DATABASE ltc_demo FROM PUBLIC;
GRANT CONNECT ON DATABASE ltc_demo TO ltc_demo_app;
-- 如果之後有其他角色（例如另一個維運用的唯讀角色）也需要連 ltc_demo，在這裡另外加一行
-- GRANT CONNECT ON DATABASE ltc_demo TO <角色>;，不要重新開放 PUBLIC。

--    postgres（正式）資料庫則不同：它是 Supabase 專案原本就在用的資料庫，PUBLIC CONNECT
--    很可能被 Supabase 自己的內部服務角色（例如 supabase_admin、authenticator、
--    service_role、anon、authenticated 等 PostgREST / GoTrue / Studio 用的角色）依賴，
--    在沒有先盤點清楚正式環境目前實際在用哪些角色的情況下，貿然對 postgres 執行
--    REVOKE CONNECT FROM PUBLIC 有鎖死正式環境既有服務的風險，不適合在這支腳本裡對
--    正式 Supabase 專案自動執行。下面這行單獨對 ltc_demo_app 的 REVOKE，在 PUBLIC 仍保有
--    CONNECT 的現況下不會真的擋下它，先留著只是為了「未來有人手動收回 postgres 的
--    PUBLIC CONNECT 後」能立即生效；在那之前，正式資料庫這一側的實際防線是
--    ltc_demo_app 的密碼只會被放進 Demo Cloud Run 服務的 DATABASE_URL 密鑰、
--    不會出現在正式服務的任何設定裡，加上既有的 JWT data_plane claim 檢查。
--    TODO（人工後續事項）：盤點正式環境目前依賴 PUBLIC CONNECT 的角色清單後，
--    比照上面 ltc_demo 的做法，對 postgres 也收回 PUBLIC CONNECT 並逐一補回合法角色。
REVOKE CONNECT ON DATABASE postgres FROM ltc_demo_app;
