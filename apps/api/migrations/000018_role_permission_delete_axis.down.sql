UPDATE roles
SET permissions = (
	SELECT COALESCE(
		jsonb_object_agg(module_key, module_value - 'delete'),
		'{}'::jsonb
	)
	FROM jsonb_each(roles.permissions) AS t(module_key, module_value)
);
