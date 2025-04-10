-- Rename existing tables with mcmp_ prefix
ALTER TABLE users RENAME TO mcmp_users;
ALTER TABLE menus RENAME TO mcmp_menu;

-- Rename roles to mcmp_platform_roles and user_roles to mcmp_user_platform_roles
ALTER TABLE roles RENAME TO mcmp_platform_roles;
ALTER TABLE user_roles RENAME TO mcmp_user_platform_roles;
-- Adjust FK column name in mcmp_user_platform_roles
ALTER TABLE mcmp_user_platform_roles RENAME COLUMN role_id TO platform_role_id;
-- Adjust FK constraint (assuming default naming convention, check actual name if error)
-- Example: ALTER TABLE mcmp_user_platform_roles DROP CONSTRAINT user_roles_role_id_fkey;
-- Example: ALTER TABLE mcmp_user_platform_roles ADD CONSTRAINT mcmp_user_platform_roles_platform_role_id_fkey FOREIGN KEY (platform_role_id) REFERENCES mcmp_platform_roles(id) ON DELETE CASCADE;

-- Create new workspace roles table
CREATE TABLE IF NOT EXISTS mcmp_workspace_roles (
    id SERIAL PRIMARY KEY,
    workspace_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (workspace_id, name)
);

-- Create new user-workspace roles mapping table
CREATE TABLE IF NOT EXISTS mcmp_user_workspace_roles (
    user_id INTEGER REFERENCES mcmp_users(id) ON DELETE CASCADE,
    workspace_role_id INTEGER REFERENCES mcmp_workspace_roles(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, workspace_role_id)
);

-- Create new permissions table
CREATE TABLE IF NOT EXISTS mcmp_permissions (
    id VARCHAR(255) PRIMARY KEY,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Rename role_menus to mcmp_role_permissions and modify structure
ALTER TABLE role_menus RENAME TO mcmp_role_permissions;
ALTER TABLE mcmp_role_permissions ADD COLUMN role_type VARCHAR(50) NOT NULL DEFAULT 'platform'; -- Add role_type, default to platform for existing data
ALTER TABLE mcmp_role_permissions RENAME COLUMN menu_id TO permission_id;
ALTER TABLE mcmp_role_permissions ALTER COLUMN permission_id TYPE VARCHAR(255); -- Change type if menu_id was INTEGER
-- Drop old primary key and foreign keys (assuming default names)
-- Example: ALTER TABLE mcmp_role_permissions DROP CONSTRAINT role_menus_pkey;
-- Example: ALTER TABLE mcmp_role_permissions DROP CONSTRAINT role_menus_role_id_fkey;
-- Example: ALTER TABLE mcmp_role_permissions DROP CONSTRAINT role_menus_menu_id_fkey;
-- Add new primary key
ALTER TABLE mcmp_role_permissions ADD PRIMARY KEY (role_type, role_id, permission_id);
-- Add new foreign keys (assuming role_id refers to platform_roles for existing data)
-- Example: ALTER TABLE mcmp_role_permissions ADD CONSTRAINT mcmp_role_permissions_platform_role_id_fkey FOREIGN KEY (role_id) REFERENCES mcmp_platform_roles(id) ON DELETE CASCADE;
-- Example: ALTER TABLE mcmp_role_permissions ADD CONSTRAINT mcmp_role_permissions_permission_id_fkey FOREIGN KEY (permission_id) REFERENCES mcmp_permissions(id) ON DELETE CASCADE;

-- Note: Adjusting FKs and defaults requires careful handling based on existing data and constraints.
-- The SQL above provides a template; actual execution might need adjustments based on DB state and constraint names.
