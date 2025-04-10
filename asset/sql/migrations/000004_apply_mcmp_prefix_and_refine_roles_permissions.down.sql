-- Drop potentially added foreign keys (use actual names from your DB)
-- Example: ALTER TABLE mcmp_role_permissions DROP CONSTRAINT IF EXISTS mcmp_role_permissions_platform_role_id_fkey;
-- Example: ALTER TABLE mcmp_role_permissions DROP CONSTRAINT IF EXISTS mcmp_role_permissions_permission_id_fkey;
-- Example: ALTER TABLE mcmp_user_platform_roles DROP CONSTRAINT IF EXISTS mcmp_user_platform_roles_platform_role_id_fkey;

-- Drop new tables
DROP TABLE IF EXISTS mcmp_user_workspace_roles;
DROP TABLE IF EXISTS mcmp_workspace_roles;
DROP TABLE IF EXISTS mcmp_permissions;

-- Revert mcmp_role_permissions back to role_menus
-- Drop new primary key
ALTER TABLE mcmp_role_permissions DROP CONSTRAINT IF EXISTS mcmp_role_permissions_pkey; -- Use actual PK name if different
-- Rename column back
ALTER TABLE mcmp_role_permissions RENAME COLUMN permission_id TO menu_id;
-- Change type back if needed (assuming menu_id was INTEGER)
-- Example: ALTER TABLE mcmp_role_permissions ALTER COLUMN menu_id TYPE INTEGER USING menu_id::integer;
-- Remove added column
ALTER TABLE mcmp_role_permissions DROP COLUMN role_type;
-- Rename table back
ALTER TABLE mcmp_role_permissions RENAME TO role_menus;
-- Re-add original primary key and foreign keys (use original names if known)
-- Example: ALTER TABLE role_menus ADD PRIMARY KEY (role_id, menu_id);
-- Example: ALTER TABLE role_menus ADD CONSTRAINT role_menus_role_id_fkey FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE; -- Note: roles table doesn't exist yet
-- Example: ALTER TABLE role_menus ADD CONSTRAINT role_menus_menu_id_fkey FOREIGN KEY (menu_id) REFERENCES menus(id) ON DELETE CASCADE; -- Note: menus table doesn't exist yet

-- Revert mcmp_user_platform_roles back to user_roles
ALTER TABLE mcmp_user_platform_roles RENAME COLUMN platform_role_id TO role_id;
ALTER TABLE mcmp_user_platform_roles RENAME TO user_roles;
-- Re-add original foreign key (use original name if known)
-- Example: ALTER TABLE user_roles ADD CONSTRAINT user_roles_role_id_fkey FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE; -- Note: roles table doesn't exist yet

-- Rename tables back
ALTER TABLE mcmp_platform_roles RENAME TO roles;
ALTER TABLE mcmp_menu RENAME TO menus;
ALTER TABLE mcmp_users RENAME TO users;

-- Re-add remaining original foreign keys after tables are renamed back
-- Example: ALTER TABLE user_roles ADD CONSTRAINT user_roles_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
-- Example: ALTER TABLE menus ADD CONSTRAINT menus_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES menus(id) ON DELETE CASCADE;

-- Note: Rollback involving table/column renames and constraint changes is complex.
-- Thorough testing is recommended. The safest rollback might involve restoring from a backup.
