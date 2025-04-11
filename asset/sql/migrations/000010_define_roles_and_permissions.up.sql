-- Add platformadmin role (Renamed from platform_superadmin)
INSERT INTO mcmp_platform_roles (name, description)
VALUES ('platformadmin', 'Platform Administrator with all privileges') -- Renamed role and updated description
ON CONFLICT (name) DO NOTHING;

-- Add predefined platform roles from .env (admin, operator, viewer, billadmin, billviewer)
INSERT INTO mcmp_platform_roles (name, description) VALUES
    ('admin', 'Administrator role with user management capabilities'),
    ('operator', 'Operator role with operational privileges'),
    ('viewer', 'Viewer role with read-only access'),
    ('billadmin', 'Billing administrator role'),
    ('billviewer', 'Billing viewer role')
ON CONFLICT (name) DO NOTHING;

-- Add permissions for user management and approval
INSERT INTO mcmp_permissions (id, description) VALUES
    ('user:create', 'Allow creating new users (admin only)'),
    ('user:list', 'Allow listing users'),
    ('user:get', 'Allow getting user details'),
    ('user:update', 'Allow updating user details'),
    ('user:delete', 'Allow deleting users'),
    ('user:approve', 'Allow approving user registration requests'), -- Ensure this exists
    ('user:reject', 'Allow rejecting user registration requests'), -- Ensure this exists
    ('registration:list', 'Allow listing pending registration requests') -- This might be removed if separate table is gone
ON CONFLICT (id) DO NOTHING;

-- Assign all permissions to platformadmin (Renamed from platform_superadmin)
-- Note: This assumes all relevant permissions are already in mcmp_permissions.
-- A more robust approach might involve dynamically getting all permission IDs.
INSERT INTO mcmp_role_permissions (role_type, role_id, permission_id)
SELECT 'platform', r.id, p.id
FROM mcmp_platform_roles r, mcmp_permissions p
WHERE r.name = 'platformadmin' -- Renamed role
ON CONFLICT (role_type, role_id, permission_id) DO NOTHING;

-- Assign specific user management permissions to admin role
INSERT INTO mcmp_role_permissions (role_type, role_id, permission_id)
SELECT 'platform', r.id, p.id
FROM mcmp_platform_roles r, mcmp_permissions p
WHERE r.name = 'admin'
  AND p.id IN ('user:create', 'user:list', 'user:get', 'user:update', 'user:delete', 'user:approve', 'user:reject' /*, 'registration:list' - remove if table removed */)
ON CONFLICT (role_type, role_id, permission_id) DO NOTHING;

-- Assign basic permissions (e.g., view menus) to other predefined roles if needed
-- Example for 'viewer':
-- INSERT INTO mcmp_role_permissions (role_type, role_id, permission_id)
-- SELECT 'platform', r.id, p.id
-- FROM mcmp_platform_roles r, mcmp_permissions p
-- WHERE r.name = 'viewer'
--   AND p.id IN ('dashboard', 'settings', ...) -- Add relevant menu permission IDs
-- ON CONFLICT (role_type, role_id, permission_id) DO NOTHING;
