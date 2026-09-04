-- 補回因先前 000021 撞號而未被套用的 settings_holidays 與 ops_tasks 權限，
-- 並確認 settings_users 與 settings_roles 的 delete 權限。

-- 1. 補上 settings_holidays 與 ops_tasks 兩個模組權限
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

-- 2. 補回 settings_users 與 settings_roles 的 delete 權限（僅 admin 可刪除）
UPDATE roles
SET permissions = jsonb_set(
	jsonb_set(permissions, '{settings_users,delete}', to_jsonb(base_role = 'admin')),
	'{settings_roles,delete}', to_jsonb(base_role = 'admin')
)
WHERE permissions ? 'settings_users' AND permissions ? 'settings_roles';
