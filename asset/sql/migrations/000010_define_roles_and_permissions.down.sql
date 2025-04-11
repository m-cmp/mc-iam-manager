-- Remove role-permission mappings added in the up migration
DELETE FROM mcmp_role_permissions
WHERE role_type = 'platform'
  AND role_id IN (SELECT id FROM mcmp_platform_roles WHERE name IN ('platform_superadmin', 'admin'));

-- Remove permissions added in the up migration
DELETE FROM mcmp_permissions
WHERE id IN (
    'user:create', 'user:list', 'user:get', 'user:update', 'user:delete',
    'user:approve', 'user:reject', 'registration:list' -- Ensure these match the up file
);

-- Remove roles added ONLY in the up migration (platform_superadmin)
-- Keep predefined roles as they might be used elsewhere or defined earlier.
DELETE FROM mcmp_platform_roles
WHERE name = 'platform_superadmin';

-- Note: This down migration assumes predefined roles like 'admin', 'operator' etc.
-- might be used by other parts or migrations and should not be deleted here.
-- If they were exclusively added by this migration, they should be deleted too.
