-- Initial data seeding based on migrations 000005, 000009, 000010

-- Seed initial platform roles (from 000010)
INSERT INTO mcmp_platform_roles (name, description) VALUES
    ('platformadmin', 'Platform Administrator with all privileges'),
    ('admin', 'Administrator role with user management capabilities'),
    ('operator', 'Operator role with operational privileges'),
    ('viewer', 'Viewer role with read-only access'),
    ('billadmin', 'Billing administrator role'),
    ('billviewer', 'Billing viewer role')
ON CONFLICT (name) DO NOTHING;

-- Seed initial menu data (from 000005)
INSERT INTO mcmp_menu (id, parent_id, display_name, res_type, is_action, priority, menu_number)
VALUES ('dashboard', NULL, 'Dashboard', 'menu', false, 1, 1)
ON CONFLICT (id) DO NOTHING;
-- Note: Add other essential menus if they were previously seeded via YAML or other migrations

-- Seed default workspace
INSERT INTO mcmp_workspaces (name, description) VALUES
    ('default', 'Default workspace for unassigned projects')
ON CONFLICT (name) DO NOTHING; -- Assuming name should be unique

-- Seed permissions (from 000009 and 000010)
-- Note: The 'name' column was added based on model analysis, assuming it should often match 'id'. Adjust if needed.
INSERT INTO mcmp_permissions (id, name, description) VALUES
    -- Menu Permissions (assuming name is same as id for simplicity here)
    ('dashboard', 'dashboard', 'Allow viewing dashboard menu'),
    ('settings', 'settings', 'Allow viewing settings menu'),
    ('accountnaccess', 'accountnaccess', 'Allow viewing account & access menu'),
    ('organizations', 'organizations', 'Allow viewing organizations menu'),
    ('users', 'users', 'Allow viewing users menu'),
    ('operations', 'operations', 'Allow viewing operations menu'),
    ('manage', 'manage', 'Allow viewing manage menu'),
    ('workspaces', 'workspaces', 'Allow viewing workspaces menu'),
    -- User Management Permissions
    ('user:create', 'user:create', 'Allow creating new users (admin only)'),
    ('user:list', 'user:list', 'Allow listing users'),
    ('user:get', 'user:get', 'Allow getting user details'),
    ('user:update', 'user:update', 'Allow updating user details'),
    ('user:delete', 'user:delete', 'Allow deleting users'),
    ('user:approve', 'user:approve', 'Allow approving user registration requests'),
    ('user:reject', 'user:reject', 'Allow rejecting user registration requests'),
    ('registration:list', 'registration:list', 'Allow listing pending registration requests') -- Keep or remove based on final logic
    -- Add other necessary permissions...
ON CONFLICT (id) DO NOTHING;

-- Assign permissions to roles (from 000009 and 000010)
-- Assign all permissions to platformadmin
INSERT INTO mcmp_role_permissions (role_type, role_id, permission_id)
SELECT 'platform', r.id, p.id
FROM mcmp_platform_roles r, mcmp_permissions p
WHERE r.name = 'platformadmin'
ON CONFLICT (role_type, role_id, permission_id) DO NOTHING;

-- Assign specific permissions to admin
INSERT INTO mcmp_role_permissions (role_type, role_id, permission_id)
SELECT 'platform', r.id, p.id
FROM mcmp_platform_roles r, mcmp_permissions p
WHERE r.name = 'admin'
  AND p.id IN (
    -- Menu permissions for admin (example, adjust as needed)
    'dashboard', 'settings', 'accountnaccess', 'organizations', 'users', 'operations', 'manage', 'workspaces',
    -- User management permissions
    'user:create', 'user:list', 'user:get', 'user:update', 'user:delete', 'user:approve', 'user:reject', 'registration:list'
    -- Add other admin-specific permission IDs...
  )
ON CONFLICT (role_type, role_id, permission_id) DO NOTHING;

-- Assign permissions for other roles (example for 'viewer')
-- INSERT INTO mcmp_role_permissions (role_type, role_id, permission_id)
-- SELECT 'platform', r.id, p.id
-- FROM mcmp_platform_roles r, mcmp_permissions p
-- WHERE r.name = 'viewer'
--   AND p.id IN ('dashboard') -- Only dashboard view permission
-- ON CONFLICT (role_type, role_id, permission_id) DO NOTHING;
