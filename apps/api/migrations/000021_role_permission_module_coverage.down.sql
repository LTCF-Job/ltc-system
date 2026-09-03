UPDATE roles
SET permissions = jsonb_set(
	jsonb_set(permissions, '{settings_users,delete}', 'false'::jsonb),
	'{settings_roles,delete}', 'false'::jsonb
)
WHERE permissions ? 'settings_users' AND permissions ? 'settings_roles';

UPDATE roles
SET permissions = (permissions - 'settings_holidays') - 'ops_tasks';
