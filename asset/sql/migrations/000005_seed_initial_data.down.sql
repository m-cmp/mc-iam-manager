-- Remove initial seeded data
DELETE FROM mcmp_menu WHERE id = 'dashboard';

-- Remove other initial data if added in the up migration
-- Example:
-- DELETE FROM mcmp_role_permissions WHERE permission_id = 'view_dashboard' AND role_type = 'platform' AND role_id = (SELECT id FROM mcmp_platform_roles WHERE name = 'admin');
-- DELETE FROM mcmp_permissions WHERE id = 'view_dashboard';
-- DELETE FROM mcmp_platform_roles WHERE name = 'admin';
