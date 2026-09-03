-- 權限矩陣加入 delete 軸：現有 view/edit 兩軸無法表達「編輯開放給 staff，但刪除僅限 admin」
-- 這種既有 API 已經在用的差異（例如個案主檔），此遷移把既有路由的實際刪除門檻回填成資料，
-- 讓 API 授權中介層改用權限矩陣時行為與遷移前一致。
UPDATE roles
SET permissions = (
	SELECT COALESCE(
		jsonb_object_agg(
			module_key,
			module_value || jsonb_build_object(
				'delete',
				CASE
					-- 這三個模組的既有 DELETE 路由要求 staff/admin（與 edit 同層級），非僅限 admin
					WHEN module_key IN ('driver_reports', 'vehicles_maintenance', 'attendance_fuel')
						THEN COALESCE((module_value ->> 'edit')::boolean, false)
					-- 其餘有 DELETE 路由的模組，既有路由僅允許 admin
					WHEN module_key IN ('masters_regions', 'masters_cases', 'masters_sites', 'masters_vehicles', 'masters_drivers', 'masters_caregivers', 'settings_notifications')
						THEN roles.base_role = 'admin'
					ELSE false
				END
			)
		),
		'{}'::jsonb
	)
	FROM jsonb_each(roles.permissions) AS t(module_key, module_value)
);
