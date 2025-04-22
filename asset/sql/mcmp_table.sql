-- Generated based on Go models in src/model/

-- Drop existing tables if they exist (optional, for clean setup)
DROP TABLE IF EXISTS mcmp_user_workspace_roles CASCADE;
DROP TABLE IF EXISTS mcmp_user_platform_roles CASCADE;
DROP TABLE IF EXISTS mcmp_role_permissions CASCADE;
DROP TABLE IF EXISTS mcmp_workspace_projects CASCADE;
DROP TABLE IF EXISTS mcmp_menu CASCADE;
DROP TABLE IF EXISTS mcmp_permissions CASCADE;
DROP TABLE IF EXISTS mcmp_platform_roles CASCADE;
DROP TABLE IF EXISTS mcmp_workspace_roles CASCADE;
DROP TABLE IF EXISTS mcmp_projects CASCADE;
DROP TABLE IF EXISTS mcmp_workspaces CASCADE;
DROP TABLE IF EXISTS mcmp_users CASCADE;
DROP TABLE IF EXISTS mcmp_token CASCADE;
DROP TABLE IF EXISTS mcmp_api_actions CASCADE;
DROP TABLE IF EXISTS mcmp_api_services CASCADE;

-- mcmp_users table
CREATE TABLE mcmp_users (
    id SERIAL PRIMARY KEY,
    kc_id VARCHAR(255) NOT NULL UNIQUE,
    username VARCHAR(255) NOT NULL UNIQUE,
    description VARCHAR(1000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- mcmp_platform_roles table
CREATE TABLE mcmp_platform_roles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    description VARCHAR(1000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- mcmp_workspace_roles table
CREATE TABLE mcmp_workspace_roles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    description VARCHAR(1000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- mcmp_user_platform_roles join table
CREATE TABLE mcmp_user_platform_roles (
    user_id INT NOT NULL,
    platform_role_id INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, platform_role_id),
    FOREIGN KEY (user_id) REFERENCES mcmp_users(id) ON DELETE CASCADE,
    FOREIGN KEY (platform_role_id) REFERENCES mcmp_platform_roles(id) ON DELETE CASCADE
);

-- mcmp_workspaces table
CREATE TABLE mcmp_workspaces (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- mcmp_user_workspace_roles join table
CREATE TABLE mcmp_user_workspace_roles (
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
CREATE TABLE mcmp_permissions (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description VARCHAR(1000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- mcmp_role_permissions join table
-- Note: FK constraint on role_id based on role_type is typically handled by application logic.
CREATE TABLE mcmp_role_permissions (
    role_type VARCHAR(50) NOT NULL,
    role_id INT NOT NULL,
    permission_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (role_type, role_id, permission_id),
    FOREIGN KEY (permission_id) REFERENCES mcmp_permissions(id) ON DELETE CASCADE
);

-- mcmp_menu table
CREATE TABLE mcmp_menu (
    id VARCHAR(255) PRIMARY KEY,
    parent_id VARCHAR(255),
    display_name VARCHAR(255) NOT NULL,
    res_type VARCHAR(255) NOT NULL,
    is_action BOOLEAN DEFAULT false,
    priority INT NOT NULL,
    menu_number INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Deferrable FK constraint for self-referencing parent_id
    CONSTRAINT fk_menu_parent FOREIGN KEY (parent_id) REFERENCES mcmp_menu(id) ON DELETE SET NULL DEFERRABLE INITIALLY DEFERRED
);

-- mcmp_projects table
CREATE TABLE mcmp_projects (
    id SERIAL PRIMARY KEY,
    nsid VARCHAR(255),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- mcmp_workspace_projects join table
CREATE TABLE mcmp_workspace_projects (
    workspace_id INT NOT NULL,
    project_id INT NOT NULL,
    PRIMARY KEY (workspace_id, project_id),
    FOREIGN KEY (workspace_id) REFERENCES mcmp_workspaces(id) ON DELETE CASCADE,
    FOREIGN KEY (project_id) REFERENCES mcmp_projects(id) ON DELETE CASCADE
);

-- mcmp_token table
CREATE TABLE mcmp_token (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    token TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- mcmp_api_services table
CREATE TABLE mcmp_api_services (
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
CREATE TABLE mcmp_api_actions (
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
CREATE INDEX idx_mcmp_api_actions_service_name ON mcmp_api_actions(service_name);

-- Trigger function to update updated_at columns (PostgreSQL specific)
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
   NEW.updated_at = NOW();
   RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply trigger to tables with updated_at
CREATE TRIGGER update_mcmp_users_updated_at BEFORE UPDATE ON mcmp_users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_mcmp_platform_roles_updated_at BEFORE UPDATE ON mcmp_platform_roles FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_mcmp_workspace_roles_updated_at BEFORE UPDATE ON mcmp_workspace_roles FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_mcmp_permissions_updated_at BEFORE UPDATE ON mcmp_permissions FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_mcmp_menu_updated_at BEFORE UPDATE ON mcmp_menu FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_mcmp_workspaces_updated_at BEFORE UPDATE ON mcmp_workspaces FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_mcmp_projects_updated_at BEFORE UPDATE ON mcmp_projects FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_mcmp_token_updated_at BEFORE UPDATE ON mcmp_token FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_mcmp_api_services_updated_at BEFORE UPDATE ON mcmp_api_services FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_mcmp_api_actions_updated_at BEFORE UPDATE ON mcmp_api_actions FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
