-- Generated based on Go models in src/model/

-- Drop existing tables if they exist (optional, for clean setup)
DROP TABLE IF EXISTS mcmp_user_workspace_roles CASCADE;
DROP TABLE IF EXISTS mcmp_user_platform_roles CASCADE;
DROP TABLE IF EXISTS mcmp_mciam_role_permissions CASCADE; -- Updated table name
DROP TABLE IF EXISTS mcmp_role_permissions CASCADE; -- Keep old name just in case for cleanup
DROP TABLE IF EXISTS mcmp_workspace_projects CASCADE;
DROP TABLE IF EXISTS mcmp_menu CASCADE;
DROP TABLE IF EXISTS mcmp_mciam_permissions CASCADE; -- Updated table name
DROP TABLE IF EXISTS mcmp_permissions CASCADE; -- Keep old name just in case for cleanup
DROP TABLE IF EXISTS mcmp_platform_roles CASCADE;
DROP TABLE IF EXISTS mcmp_workspace_roles CASCADE;
DROP TABLE IF EXISTS mcmp_projects CASCADE;
DROP TABLE IF EXISTS mcmp_workspaces CASCADE;
DROP TABLE IF EXISTS mcmp_users CASCADE;
DROP TABLE IF EXISTS mcmp_token CASCADE;
DROP TABLE IF EXISTS mcmp_api_actions CASCADE;
DROP TABLE IF EXISTS mcmp_api_services CASCADE;
-- Drop old/incorrect CSP mapping tables
DROP TABLE IF EXISTS mcmp_csp_permissions CASCADE;
DROP TABLE IF EXISTS mciam_role_csp_permissions CASCADE;
DROP TABLE IF EXISTS mcmp_role_csp_permissions CASCADE;
DROP TABLE IF EXISTS mcmp_csp_role_permission CASCADE;
DROP TABLE IF EXISTS mcmp_mciam_role_csp_role_mapping CASCADE;
DROP TABLE IF EXISTS mcmp_workspace_role_csp_role_mapping CASCADE;


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

-- mcmp_resource_types table
CREATE TABLE mcmp_resource_types (
    framework_id VARCHAR(100) NOT NULL, -- Identifier of the framework (e.g., "mc-iam-manager", "mc-infra-manager")
    id VARCHAR(100) NOT NULL,          -- Unique identifier within the framework (e.g., "workspace", "vm")
    name VARCHAR(255) NOT NULL,        -- Display name (e.g., "Workspace", "Virtual Machine")
    description VARCHAR(1000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (framework_id, id)
);

-- mcmp_mciam_permissions table (Renamed)
CREATE TABLE mcmp_mciam_permissions (
    id VARCHAR(255) PRIMARY KEY, -- Format: <framework_id>:<resource_type_id>:<action>
    framework_id VARCHAR(100) NOT NULL,
    resource_type_id VARCHAR(100) NOT NULL,
    action VARCHAR(100) NOT NULL, -- e.g., create, read, update, delete
    name VARCHAR(100) NOT NULL,
    description VARCHAR(1000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (framework_id, resource_type_id) REFERENCES mcmp_resource_types(framework_id, id) ON DELETE CASCADE -- Cascade delete permissions if resource type is deleted
);

-- mcmp_mciam_role_permissions join table (Added)
CREATE TABLE mcmp_mciam_role_permissions (
    role_type VARCHAR(50) NOT NULL, -- Should likely always be 'workspace' for this table
    role_id INT NOT NULL, -- Renamed column for clarity
    permission_id VARCHAR(255) NOT NULL, -- FK to mcmp_mciam_permissions.id
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (role_type, role_id, permission_id), -- Updated PK columns
    FOREIGN KEY (permission_id) REFERENCES mcmp_mciam_permissions(id) ON DELETE CASCADE, -- Updated FK target table
    FOREIGN KEY (workspace_role_id) REFERENCES mcmp_workspace_roles(id) ON DELETE CASCADE -- Added FK to workspace roles
);

-- mcmp_workspace_role_csp_role_mapping table (Corrected Name)
CREATE TABLE mcmp_workspace_role_csp_role_mapping (
    workspace_role_id INT NOT NULL, 		-- FK to mcmp_workspace_roles.id
    csp_type VARCHAR(50) NOT NULL, 			-- e.g., "aws", "gcp", "azure"
    csp_role_arn VARCHAR(255) NOT NULL, 	-- The actual ARN or identifier of the role in the CSP
    idp_identifier VARCHAR(255), 			-- e.g., AWS OIDC Provider ARN (Nullable)
    description VARCHAR(1000), 				-- Description of this specific mapping (Nullable)
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- No updated_at needed for mapping table
    PRIMARY KEY (workspace_role_id, csp_type, csp_role_arn), -- Composite primary key
    FOREIGN KEY (workspace_role_id) REFERENCES mcmp_workspace_roles(id) ON DELETE CASCADE
    -- No FK for csp_role_arn as it's external identifier
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
CREATE TRIGGER update_mcmp_resource_types_updated_at BEFORE UPDATE ON mcmp_resource_types FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_mcmp_mciam_permissions_updated_at BEFORE UPDATE ON mcmp_mciam_permissions FOR EACH ROW EXECUTE FUNCTION update_updated_at_column(); -- Updated trigger table name
-- Remove trigger for deleted mcmp_csp_permissions table
-- CREATE TRIGGER update_mcmp_csp_permissions_updated_at BEFORE UPDATE ON mcmp_csp_permissions FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_mcmp_menu_updated_at BEFORE UPDATE ON mcmp_menu FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_mcmp_workspaces_updated_at BEFORE UPDATE ON mcmp_workspaces FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_mcmp_projects_updated_at BEFORE UPDATE ON mcmp_projects FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_mcmp_token_updated_at BEFORE UPDATE ON mcmp_token FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_mcmp_api_services_updated_at BEFORE UPDATE ON mcmp_api_services FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_mcmp_api_actions_updated_at BEFORE UPDATE ON mcmp_api_actions FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- mcmp_mciam_permission_action_mappings table
CREATE TABLE mcmp_mciam_permission_action_mappings (
    id SERIAL PRIMARY KEY,
    permission_id VARCHAR(255) NOT NULL,  -- mcmp_mciam_permissions 테이블의 id 참조
    action_id INTEGER NOT NULL,           -- mcmp_api_actions 테이블의 id 참조
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (permission_id) REFERENCES mcmp_mciam_permissions(id) ON DELETE CASCADE,
    FOREIGN KEY (action_id) REFERENCES mcmp_api_actions(id) ON DELETE CASCADE,
    UNIQUE (permission_id, action_id)     -- 중복 매핑 방지
);

-- MCMP API 권한-액션 매핑 테이블
CREATE TABLE IF NOT EXISTS mcmp_api_permission_action_mappings (
    id SERIAL PRIMARY KEY,
    permission_id VARCHAR(255) NOT NULL,
    action_id INTEGER NOT NULL,
    action_name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(permission_id, action_id)
);

-- Trigger for mcmp_mciam_permission_action_mappings
CREATE TRIGGER update_mcmp_mciam_permission_action_mappings_updated_at 
BEFORE UPDATE ON mcmp_mciam_permission_action_mappings 
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create menu mapping table
CREATE TABLE IF NOT EXISTS mcmp_platform_role_menu_mappings (
    id SERIAL PRIMARY KEY,
    platform_role VARCHAR(100) NOT NULL,
    menu_id VARCHAR(100) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(platform_role, menu_id)
);

-- Add foreign key constraints
ALTER TABLE mcmp_platform_role_menu_mappings
    ADD CONSTRAINT fk_platform_role
    FOREIGN KEY (platform_role)
    REFERENCES mcmp_platform_roles(name)
    ON DELETE CASCADE;

ALTER TABLE mcmp_platform_role_menu_mappings
    ADD CONSTRAINT fk_menu_id
    FOREIGN KEY (menu_id)
    REFERENCES mcmp_menu(id)
    ON DELETE CASCADE;

-- Create index for faster lookups
CREATE INDEX idx_platform_role_menu_mappings_platform_role
    ON mcmp_platform_role_menu_mappings(platform_role);

CREATE INDEX idx_platform_role_menu_mappings_menu_id
    ON mcmp_platform_role_menu_mappings(menu_id); 