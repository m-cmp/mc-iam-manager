-- Seed permissions for menu access (using menu IDs as permission IDs for simplicity)
-- Add entries for all relevant menu IDs from asset/menu/menu.yaml
INSERT INTO mcmp_permissions (id, description) VALUES
    ('dashboard', 'Allow viewing dashboard menu'),
    ('settings', 'Allow viewing settings menu'),
    ('accountnaccess', 'Allow viewing account & access menu'),
    ('organizations', 'Allow viewing organizations menu'),
    ('users', 'Allow viewing users menu'),
    ('operations', 'Allow viewing operations menu'),
    ('manage', 'Allow viewing manage menu'),
    ('workspaces', 'Allow viewing workspaces menu')
    -- Add permissions for all other menu items...
ON CONFLICT (id) DO NOTHING;

-- Assign menu permissions to the 'admin' platform role (example)
-- Assumes 'admin' role exists in mcmp_platform_roles (created manually or by previous migration)
INSERT INTO mcmp_role_permissions (role_type, role_id, permission_id)
SELECT 'platform', r.id, p.id
FROM mcmp_platform_roles r, mcmp_permissions p
WHERE r.name = 'admin'
  AND p.id IN ('dashboard', 'settings', 'accountnaccess', 'organizations', 'users', 'operations', 'manage', 'workspaces' /*, other admin-accessible menu IDs... */)
ON CONFLICT (role_type, role_id, permission_id) DO NOTHING;

-- Assign permissions for other roles (e.g., 'viewer') if needed
-- INSERT INTO mcmp_role_permissions (role_type, role_id, permission_id)
-- SELECT 'platform', r.id, p.id
-- FROM mcmp_platform_roles r, mcmp_permissions p
-- WHERE r.name = 'viewer'
--   AND p.id IN ('dashboard' /*, other viewer-accessible menu IDs... */)
-- ON CONFLICT (role_type, role_id, permission_id) DO NOTHING;
