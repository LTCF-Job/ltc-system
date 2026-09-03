-- 使用者／角色／國定假日／維運任務這四類端點原本以 auth.RequireRoles 寫死角色字面值授權，
-- 管理員自建的角色永遠不符合比對而被擋。改走權限矩陣後，這裡把既有路由的實際門檻回填成
-- 資料，讓所有角色（含自訂角色）在遷移前後拿到一致的存取範圍。

-- 1. 補上 settings_holidays 與 ops_tasks 兩個新模組。
--    settings_holidays：GET 原本開放 viewer/staff/admin、POST 給 staff/admin、DELETE 僅 admin。
--    ops_tasks：兩支手動觸發端點原本都要求 staff/admin，屬異動操作，故 view 與 edit 同門檻、
--    無 DELETE 路由故 delete 一律 false。
UPDATE roles
SET permissions = permissions || jsonb_build_object(
	'settings_holidays', jsonb_build_object(
		'view', true,
		'edit', base_role IN ('staff', 'admin'),
		'delete', base_role = 'admin'
	),
	'ops_tasks', jsonb_build_object(
		'view', base_role IN ('staff', 'admin'),
		'edit', base_role IN ('staff', 'admin'),
		'delete', false
	)
);

-- 2. 補回 settings_users 與 settings_roles 的 delete：000018 的兩份模組清單都漏列這兩個，
--    使其落入 ELSE false，導致 DELETE /users/:id 與 DELETE /roles/:id 改用 delete 軸後連
--    管理員都會被擋。門檻沿用 000018「僅 admin」那組的寫法。
UPDATE roles
SET permissions = jsonb_set(
	jsonb_set(permissions, '{settings_users,delete}', to_jsonb(base_role = 'admin')),
	'{settings_roles,delete}', to_jsonb(base_role = 'admin')
)
WHERE permissions ? 'settings_users' AND permissions ? 'settings_roles';
