-- Generated based on Go models in src/model/ and initial data seeding

-- mcmp_users table
CREATE TABLE IF NOT EXISTS mcmp_users (
    id SERIAL PRIMARY KEY,
    kc_id VARCHAR(255) NOT NULL UNIQUE,
    username VARCHAR(255) NOT NULL UNIQUE,
    description VARCHAR(1000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- mcmp_platform_roles table
CREATE TABLE IF NOT EXISTS mcmp_platform_roles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    description VARCHAR(1000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- mcmp_workspace_roles table
CREATE TABLE IF NOT EXISTS mcmp_workspace_roles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    description VARCHAR(1000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- mcmp_user_platform_roles join table
CREATE TABLE IF NOT EXISTS mcmp_user_platform_roles (
    user_id INT NOT NULL,
    platform_role_id INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, platform_role_id),
    FOREIGN KEY (user_id) REFERENCES mcmp_users(id) ON DELETE CASCADE,
    FOREIGN KEY (platform_role_id) REFERENCES mcmp_platform_roles(id) ON DELETE CASCADE
);

-- mcmp_workspaces table
CREATE TABLE IF NOT EXISTS mcmp_workspaces (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- mcmp_user_workspace_roles join table
CREATE TABLE IF NOT EXISTS mcmp_user_workspace_roles (
    user_id INT NOT NULL,
    workspace_id INT NOT NULL,
    workspace_role_id INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, workspace_id, workspace_role_id),
    FOREIGN KEY (user_id) REFERENCES mcmp_users(id) ON DELETE CASCADE,
    FOREIGN KEY (workspace_id) REFERENCES mcmp_workspaces(id) ON DELETE CASCADE,
    FOREIGN KEY (workspace_role_id) REFERENCES mcmp_workspace_roles(id) ON DELETE CASCADE
);

-- mcmp_permissions table
CREATE TABLE IF NOT EXISTS mcmp_permissions (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description VARCHAR(1000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- mcmp_role_permissions join table
CREATE TABLE IF NOT EXISTS mcmp_role_permissions (
    role_type VARCHAR(50) NOT NULL,
    role_id INT NOT NULL,
    permission_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (role_type, role_id, permission_id),
    FOREIGN KEY (permission_id) REFERENCES mcmp_permissions(id) ON DELETE CASCADE
);

-- mcmp_menu table
CREATE TABLE IF NOT EXISTS mcmp_menu (
    id VARCHAR(255) PRIMARY KEY,
    parent_id VARCHAR(255),
    display_name VARCHAR(255) NOT NULL,
    res_type VARCHAR(255) NOT NULL,
    is_action BOOLEAN DEFAULT false,
    priority INT NOT NULL,
    menu_number INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_menu_parent FOREIGN KEY (parent_id) REFERENCES mcmp_menu(id) ON DELETE SET NULL DEFERRABLE INITIALLY DEFERRED
);

-- mcmp_projects table
CREATE TABLE IF NOT EXISTS mcmp_projects (
    id SERIAL PRIMARY KEY,
    nsid VARCHAR(255),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- mcmp_workspace_projects join table
CREATE TABLE IF NOT EXISTS mcmp_workspace_projects (
    workspace_id INT NOT NULL,
    project_id INT NOT NULL,
    PRIMARY KEY (workspace_id, project_id),
    FOREIGN KEY (workspace_id) REFERENCES mcmp_workspaces(id) ON DELETE CASCADE,
    FOREIGN KEY (project_id) REFERENCES mcmp_projects(id) ON DELETE CASCADE
);

-- mcmp_token table
CREATE TABLE IF NOT EXISTS mcmp_token (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    token TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- mcmp_api_services table
CREATE TABLE IF NOT EXISTS mcmp_api_services (
    name VARCHAR(100) PRIMARY KEY,
    version VARCHAR(50),
    base_url VARCHAR(255),
    auth_type VARCHAR(50),
    auth_user VARCHAR(100),
    auth_pass VARCHAR(255),
    is_active BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- mcmp_api_actions table
CREATE TABLE IF NOT EXISTS mcmp_api_actions (
    id SERIAL PRIMARY KEY,
    service_name VARCHAR(100) NOT NULL,
    action_name VARCHAR(100) NOT NULL,
    method VARCHAR(10) NOT NULL,
    resource_path VARCHAR(500),
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (service_name) REFERENCES mcmp_api_services(name) ON DELETE CASCADE
);

-- Index for service_name in mcmp_api_actions
CREATE INDEX IF NOT EXISTS idx_mcmp_api_actions_service_name ON mcmp_api_actions(service_name);

-- Trigger function to update updated_at columns (PostgreSQL specific)
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
   NEW.updated_at = NOW();
   RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply trigger to tables with updated_at (Use CREATE OR REPLACE TRIGGER if supported, otherwise check existence)
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_mcmp_users_updated_at') THEN
        CREATE TRIGGER update_mcmp_users_updated_at BEFORE UPDATE ON mcmp_users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_mcmp_platform_roles_updated_at') THEN
        CREATE TRIGGER update_mcmp_platform_roles_updated_at BEFORE UPDATE ON mcmp_platform_roles FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_mcmp_workspace_roles_updated_at') THEN
        CREATE TRIGGER update_mcmp_workspace_roles_updated_at BEFORE UPDATE ON mcmp_workspace_roles FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_mcmp_permissions_updated_at') THEN
        CREATE TRIGGER update_mcmp_permissions_updated_at BEFORE UPDATE ON mcmp_permissions FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_mcmp_menu_updated_at') THEN
        CREATE TRIGGER update_mcmp_menu_updated_at BEFORE UPDATE ON mcmp_menu FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_mcmp_workspaces_updated_at') THEN
        CREATE TRIGGER update_mcmp_workspaces_updated_at BEFORE UPDATE ON mcmp_workspaces FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_mcmp_projects_updated_at') THEN
        CREATE TRIGGER update_mcmp_projects_updated_at BEFORE UPDATE ON mcmp_projects FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_mcmp_token_updated_at') THEN
        CREATE TRIGGER update_mcmp_token_updated_at BEFORE UPDATE ON mcmp_token FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_mcmp_api_services_updated_at') THEN
        CREATE TRIGGER update_mcmp_api_services_updated_at BEFORE UPDATE ON mcmp_api_services FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_mcmp_api_actions_updated_at') THEN
        CREATE TRIGGER update_mcmp_api_actions_updated_at BEFORE UPDATE ON mcmp_api_actions FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
END $$;


-- Seed initial platform roles
INSERT INTO mcmp_platform_roles (name, description) VALUES
    ('platformadmin', 'Platform Administrator with all privileges'),
    ('admin', 'Administrator role with user management capabilities'),
    ('operator', 'Operator role with operational privileges'),
    ('viewer', 'Viewer role with read-only access'),
    ('billadmin', 'Billing administrator role'),
    ('billviewer', 'Billing viewer role')
ON CONFLICT (name) DO NOTHING;

-- Seed initial menu data
INSERT INTO mcmp_menu (id, parent_id, display_name, res_type, is_action, priority, menu_number)
VALUES ('dashboard', NULL, 'Dashboard', 'menu', false, 1, 1)
ON CONFLICT (id) DO NOTHING;

-- Seed default workspace
INSERT INTO mcmp_workspaces (name, description) VALUES
    ('default', 'Default workspace for unassigned projects')
ON CONFLICT (name) DO NOTHING; -- Assuming name should be unique

-- Seed permissions
INSERT INTO mcmp_permissions (id, name, description) VALUES
    ('dashboard', 'dashboard', 'Allow viewing dashboard menu'),
    ('settings', 'settings', 'Allow viewing settings menu'),
    ('accountnaccess', 'accountnaccess', 'Allow viewing account & access menu'),
    ('organizations', 'organizations', 'Allow viewing organizations menu'),
    ('users', 'users', 'Allow viewing users menu'),
    ('operations', 'operations', 'Allow viewing operations menu'),
    ('manage', 'manage', 'Allow viewing manage menu'),
    ('workspaces', 'workspaces', 'Allow viewing workspaces menu'),
    ('user:create', 'user:create', 'Allow creating new users (admin only)'),
    ('user:list', 'user:list', 'Allow listing users'),
    ('user:get', 'user:get', 'Allow getting user details'),
    ('user:update', 'user:update', 'Allow updating user details'),
    ('user:delete', 'user:delete', 'Allow deleting users'),
    ('user:approve', 'user:approve', 'Allow approving user registration requests'),
    ('user:reject', 'user:reject', 'Allow rejecting user registration requests'),
    ('registration:list', 'registration:list', 'Allow listing pending registration requests')
ON CONFLICT (id) DO NOTHING;

-- Assign permissions to roles
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
    'dashboard', 'settings', 'accountnaccess', 'organizations', 'users', 'operations', 'manage', 'workspaces',
    'user:create', 'user:list', 'user:get', 'user:update', 'user:delete', 'user:approve', 'user:reject', 'registration:list'
  )
ON CONFLICT (role_type, role_id, permission_id) DO NOTHING;
