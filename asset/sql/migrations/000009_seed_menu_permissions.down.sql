-- Remove role-permission mappings for menus added in the up migration
DELETE FROM mcmp_role_permissions
WHERE role_type = 'platform'
  AND permission_id IN (
    'dashboard', 'settings', 'accountnaccess', 'organizations', 'users',
    'operations', 'manage', 'workspaces' /*, other menu IDs... */
  );

-- Remove menu permissions from the permissions table
DELETE FROM mcmp_permissions
WHERE id IN (
    'dashboard', 'settings', 'accountnaccess', 'organizations', 'users',
    'operations', 'manage', 'workspaces' /*, other menu IDs... */
  );
