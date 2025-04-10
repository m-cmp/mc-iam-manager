-- Seed initial menu data
INSERT INTO mcmp_menu (id, parent_id, display_name, res_type, is_action, priority, menu_number)
VALUES ('dashboard', NULL, 'Dashboard', 'menu', false, 1, 1)
ON CONFLICT (id) DO NOTHING;

-- Add other initial data seeding here if needed (e.g., default platform roles, permissions)
-- Example:
-- INSERT INTO mcmp_platform_roles (name, description) VALUES ('admin', 'Platform Administrator') ON CONFLICT (name) DO NOTHING;
-- INSERT INTO mcmp_permissions (id, description) VALUES ('view_dashboard', 'Can view dashboard') ON CONFLICT (id) DO NOTHING;
-- INSERT INTO mcmp_role_permissions (role_type, role_id, permission_id)
-- SELECT 'platform', id, 'view_dashboard' FROM mcmp_platform_roles WHERE name = 'admin'
-- ON CONFLICT (role_type, role_id, permission_id) DO NOTHING;
