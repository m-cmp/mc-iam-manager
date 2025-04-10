-- Add mcmp_ prefix to existing tables and rename menus to mcmp_menu
ALTER TABLE users RENAME TO mcmp_users;
ALTER TABLE roles RENAME TO mcmp_roles;
ALTER TABLE user_roles RENAME TO mcmp_user_roles;
ALTER TABLE menus RENAME TO mcmp_menu;
ALTER TABLE role_menus RENAME TO mcmp_role_menus;

-- Adjust foreign key constraints if necessary (example for user_roles)
-- Note: Constraint names might differ, check your DB schema if errors occur.
-- Example: ALTER TABLE mcmp_user_roles DROP CONSTRAINT user_roles_user_id_fkey;
-- Example: ALTER TABLE mcmp_user_roles ADD CONSTRAINT mcmp_user_roles_user_id_fkey FOREIGN KEY (user_id) REFERENCES mcmp_users(id) ON DELETE CASCADE;
-- Example: ALTER TABLE mcmp_user_roles DROP CONSTRAINT user_roles_role_id_fkey;
-- Example: ALTER TABLE mcmp_user_roles ADD CONSTRAINT mcmp_user_roles_role_id_fkey FOREIGN KEY (role_id) REFERENCES mcmp_roles(id) ON DELETE CASCADE;

-- Similar adjustments might be needed for mcmp_role_menus and mcmp_menu (parent_id)
-- Example for mcmp_menu parent_id:
-- ALTER TABLE mcmp_menu DROP CONSTRAINT menus_parent_id_fkey;
-- ALTER TABLE mcmp_menu ADD CONSTRAINT mcmp_menu_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES mcmp_menu(id) ON DELETE CASCADE;

-- Example for mcmp_role_menus:
-- ALTER TABLE mcmp_role_menus DROP CONSTRAINT role_menus_role_id_fkey;
-- ALTER TABLE mcmp_role_menus ADD CONSTRAINT mcmp_role_menus_role_id_fkey FOREIGN KEY (role_id) REFERENCES mcmp_roles(id) ON DELETE CASCADE;
-- ALTER TABLE mcmp_role_menus DROP CONSTRAINT role_menus_menu_id_fkey;
-- ALTER TABLE mcmp_role_menus ADD CONSTRAINT mcmp_role_menus_menu_id_fkey FOREIGN KEY (menu_id) REFERENCES mcmp_menu(id) ON DELETE CASCADE;

-- Note: The foreign key adjustments above are examples.
-- The exact commands depend on the automatically generated constraint names in your PostgreSQL database.
-- You might need to inspect the schema (e.g., using \d table_name in psql) to find the correct constraint names before dropping and adding them.
-- If the migration fails due to constraint issues, manually adjust the constraint names in this file based on your database schema.
