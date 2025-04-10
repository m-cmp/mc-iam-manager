-- Remove mcmp_ prefix from tables and rename mcmp_menu back to menus

-- Adjust foreign key constraints first (reverse of up migration)
-- Note: Constraint names might differ, check your DB schema if errors occur.
-- Example: ALTER TABLE mcmp_user_roles DROP CONSTRAINT mcmp_user_roles_user_id_fkey;
-- Example: ALTER TABLE mcmp_user_roles ADD CONSTRAINT user_roles_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE; -- Note: users table doesn't exist yet at this point if renaming later
-- Example: ALTER TABLE mcmp_user_roles DROP CONSTRAINT mcmp_user_roles_role_id_fkey;
-- Example: ALTER TABLE mcmp_user_roles ADD CONSTRAINT user_roles_role_id_fkey FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE; -- Note: roles table doesn't exist yet

-- It might be safer to drop constraints, rename tables, then add constraints back with original names referencing original table names.

-- Drop potentially renamed constraints (use actual names from your DB)
-- Example: ALTER TABLE mcmp_user_roles DROP CONSTRAINT IF EXISTS mcmp_user_roles_user_id_fkey;
-- Example: ALTER TABLE mcmp_user_roles DROP CONSTRAINT IF EXISTS mcmp_user_roles_role_id_fkey;
-- Example: ALTER TABLE mcmp_menu DROP CONSTRAINT IF EXISTS mcmp_menu_parent_id_fkey;
-- Example: ALTER TABLE mcmp_role_menus DROP CONSTRAINT IF EXISTS mcmp_role_menus_role_id_fkey;
-- Example: ALTER TABLE mcmp_role_menus DROP CONSTRAINT IF EXISTS mcmp_role_menus_menu_id_fkey;


-- Rename tables back
ALTER TABLE mcmp_users RENAME TO users;
ALTER TABLE mcmp_roles RENAME TO roles;
ALTER TABLE mcmp_user_roles RENAME TO user_roles;
ALTER TABLE mcmp_menu RENAME TO menus;
ALTER TABLE mcmp_role_menus RENAME TO role_menus;

-- Re-add original foreign key constraints (use original names if known, or let PG auto-name)
-- Example: ALTER TABLE user_roles ADD CONSTRAINT user_roles_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
-- Example: ALTER TABLE user_roles ADD CONSTRAINT user_roles_role_id_fkey FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE;
-- Example: ALTER TABLE menus ADD CONSTRAINT menus_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES menus(id) ON DELETE CASCADE;
-- Example: ALTER TABLE role_menus ADD CONSTRAINT role_menus_role_id_fkey FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE;
-- Example: ALTER TABLE role_menus ADD CONSTRAINT role_menus_menu_id_fkey FOREIGN KEY (menu_id) REFERENCES menus(id) ON DELETE CASCADE;

-- Note: The foreign key adjustments are complex for rollback.
-- The safest approach often involves dropping constraints before rename and adding them after.
-- Ensure the constraint names and references are correct for your specific schema.
