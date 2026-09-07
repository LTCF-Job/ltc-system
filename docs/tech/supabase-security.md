# Supabase Data API 邊界

前端只使用 Supabase Auth 取得與更新 Session；業務資料一律由 API 透過 `DATABASE_URL` 讀寫。Migration `000028_public_table_access_lockdown` 因此對 `public` schema 的資料表啟用 RLS，並撤銷 `anon` 與 `authenticated` 的 table／sequence 權限。

部署到 Supabase 後仍應用下列查詢驗證實際狀態：

```sql
SELECT n.nspname AS schema_name, c.relname AS table_name, c.relrowsecurity
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = 'public' AND c.relkind = 'r';

SELECT schemaname, tablename, policyname, roles, cmd
FROM pg_policies
WHERE schemaname = 'public';
```

完成條件是業務表對瀏覽器使用的 `anon`／`authenticated` 不可直接存取，且 `/api/v1` 仍能透過 API database role 正常運作。
