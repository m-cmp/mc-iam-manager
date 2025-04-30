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

-- mcmp_resource_types table
CREATE TABLE IF NOT EXISTS mcmp_resource_types (
    framework_id VARCHAR(100) NOT NULL, -- Identifier of the framework (e.g., "mc-iam-manager", "mc-infra-manager")
    id VARCHAR(100) NOT NULL,          -- Unique identifier within the framework (e.g., "workspace", "vm")
    name VARCHAR(255) NOT NULL,        -- Display name (e.g., "Workspace", "Virtual Machine")
    description VARCHAR(1000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (framework_id, id)
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

-- mcmp_mciam_role_permissions join table (Renamed and FK target updated)
CREATE TABLE IF NOT EXISTS mcmp_mciam_role_permissions (
    role_type VARCHAR(50) NOT NULL, -- Should likely always be 'workspace' for this table
    workspace_role_id INT NOT NULL, -- Renamed column for clarity
    permission_id VARCHAR(255) NOT NULL, -- FK to mcmp_mciam_permissions.id
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (role_type, workspace_role_id, permission_id), -- Updated PK columns
    FOREIGN KEY (permission_id) REFERENCES mcmp_mciam_permissions(id) ON DELETE CASCADE, -- Updated FK target table
    FOREIGN KEY (workspace_role_id) REFERENCES mcmp_workspace_roles(id) ON DELETE CASCADE -- Added FK to workspace roles
);

-- Remove old/incorrect CSP mapping tables if they exist
DROP TABLE IF EXISTS mcmp_csp_permissions CASCADE;
DROP TABLE IF EXISTS mciam_role_csp_permissions CASCADE;
DROP TABLE IF EXISTS mcmp_role_csp_permissions CASCADE;
DROP TABLE IF EXISTS mcmp_csp_role_permission CASCADE;
DROP TABLE IF EXISTS mcmp_mciam_role_csp_role_mapping CASCADE; -- Drop potentially created table from previous step

-- mcmp_workspace_role_csp_role_mapping table (Corrected Name)
CREATE TABLE IF NOT EXISTS mcmp_workspace_role_csp_role_mapping (
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
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_mcmp_resource_types_updated_at') THEN
        CREATE TRIGGER update_mcmp_resource_types_updated_at BEFORE UPDATE ON mcmp_resource_types FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
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
    -- Remove trigger for deleted mcmp_csp_permissions table
    -- IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_mcmp_csp_permissions_updated_at') THEN
    --     CREATE TRIGGER update_mcmp_csp_permissions_updated_at BEFORE UPDATE ON mcmp_csp_permissions FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    -- END IF;
    -- No updated_at trigger needed for mcmp_workspace_role_csp_role_mapping join table
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

-- Seed Resource Types
INSERT INTO mcmp_resource_types (framework_id, id, name, description) VALUES
    ('mc-iam-manager', 'workspace', 'Workspace', 'Management unit for projects and users'),
    ('mc-iam-manager', 'project', 'Project', 'Namespace or similar construct'),
    ('mc-iam-manager', 'user', 'User Management', 'User administration'),
    ('mc-iam-manager', 'role', 'Role Management', 'Role administration'),
    ('mc-iam-manager', 'policy', 'Policy Management', 'Policy administration'),
    ('mc-infra-manager', 'vm', 'Virtual Machine', 'Virtual Machine resources'),
    ('mc-infra-manager', 'k8s', 'Kubernetes', 'Kubernetes cluster resources'),
    ('mc-infra-manager', 'vpc', 'Virtual Network', 'Virtual Private Cloud resources'),
    ('mc-infra-manager', 'storage', 'Storage', 'Storage resources'),
    ('mc-billing', 'billing', 'Billing', 'Billing information'),
    ('mc-monitoring', 'monitoring', 'Monitoring', 'Monitoring data')
ON CONFLICT (framework_id, id) DO NOTHING;

-- Seed Permissions (based on Resource Types and Actions)
DELETE FROM mcmp_permissions; -- Clear old permissions before seeding new ones

INSERT INTO mcmp_permissions (id, framework_id, resource_type_id, action, name, description) VALUES
    -- IAM Workspace Permissions
    ('mc-iam-manager:workspace:create', 'mc-iam-manager', 'workspace', 'create', 'Create Workspace', 'Allow creating workspaces'),
    ('mc-iam-manager:workspace:read', 'mc-iam-manager', 'workspace', 'read', 'Read Workspace', 'Allow viewing workspaces'),
    ('mc-iam-manager:workspace:update', 'mc-iam-manager', 'workspace', 'update', 'Update Workspace', 'Allow updating workspaces'),
    ('mc-iam-manager:workspace:delete', 'mc-iam-manager', 'workspace', 'delete', 'Delete Workspace', 'Allow deleting workspaces'),
    ('mc-iam-manager:workspace:assign_user', 'mc-iam-manager', 'workspace', 'assign_user', 'Assign User to Workspace', 'Allow assigning users to workspaces'),
    ('mc-iam-manager:workspace:list_all', 'mc-iam-manager', 'workspace', 'list_all', 'List All Workspaces', 'Allow listing all workspaces (Platform Admin)'),
    ('mc-iam-manager:workspace:list_assigned', 'mc-iam-manager', 'workspace', 'list_assigned', 'List Assigned Workspaces', 'Allow listing assigned workspaces'),
    -- IAM Project Permissions (Example)
    ('mc-iam-manager:project:create', 'mc-iam-manager', 'project', 'create', 'Create Project', 'Allow creating projects'),
    ('mc-iam-manager:project:read', 'mc-iam-manager', 'project', 'read', 'Read Project', 'Allow viewing projects'),
    ('mc-iam-manager:project:update', 'mc-iam-manager', 'project', 'update', 'Update Project', 'Allow updating projects'),
    ('mc-iam-manager:project:delete', 'mc-iam-manager', 'project', 'delete', 'Delete Project', 'Allow deleting projects'),
    ('mc-iam-manager:project:sync', 'mc-iam-manager', 'project', 'sync', 'Sync Projects', 'Allow syncing projects with infra'),
    -- Infra VM Permissions
    ('mc-infra-manager:vm:create', 'mc-infra-manager', 'vm', 'create', 'Create VM', 'Allow creating VMs via MCMP API'),
    ('mc-infra-manager:vm:read', 'mc-infra-manager', 'vm', 'read', 'Read VM', 'Allow reading VM info via MCMP API'),
    ('mc-infra-manager:vm:update', 'mc-infra-manager', 'vm', 'update', 'Update VM', 'Allow updating VMs via MCMP API'),
    ('mc-infra-manager:vm:delete', 'mc-infra-manager', 'vm', 'delete', 'Delete VM', 'Allow deleting VMs via MCMP API'),
    -- Billing Permissions
    ('mc-billing:billing:read', 'mc-billing', 'billing', 'read', 'Read Billing', 'Allow viewing billing information'),
    ('mc-billing:billing:update', 'mc-billing', 'billing', 'update', 'Update Billing', 'Allow updating billing settings'),
    -- Add other permissions for k8s, vpc, storage, monitoring etc.
    ('mc-infra-manager:k8s:create', 'mc-infra-manager', 'k8s', 'create', 'Create K8s', 'Allow creating K8s clusters via MCMP API'),
    ('mc-infra-manager:k8s:read', 'mc-infra-manager', 'k8s', 'read', 'Read K8s', 'Allow reading K8s info via MCMP API'),
    ('mc-infra-manager:k8s:update', 'mc-infra-manager', 'k8s', 'update', 'Update K8s', 'Allow updating K8s clusters via MCMP API'),
    ('mc-infra-manager:k8s:delete', 'mc-infra-manager', 'k8s', 'delete', 'Delete K8s', 'Allow deleting K8s clusters via MCMP API'),
    ('mc-infra-manager:vpc:create', 'mc-infra-manager', 'vpc', 'create', 'Create VPC', 'Allow creating VPCs via MCMP API'),
    ('mc-infra-manager:vpc:read', 'mc-infra-manager', 'vpc', 'read', 'Read VPC', 'Allow reading VPC info via MCMP API'),
    ('mc-infra-manager:vpc:update', 'mc-infra-manager', 'vpc', 'update', 'Update VPC', 'Allow updating VPCs via MCMP API'),
    ('mc-infra-manager:vpc:delete', 'mc-infra-manager', 'vpc', 'delete', 'Delete VPC', 'Allow deleting VPCs via MCMP API'),
    ('mc-infra-manager:storage:create', 'mc-infra-manager', 'storage', 'create', 'Create Storage', 'Allow creating Storage via MCMP API'),
    ('mc-infra-manager:storage:read', 'mc-infra-manager', 'storage', 'read', 'Read Storage', 'Allow reading Storage info via MCMP API'),
    ('mc-infra-manager:storage:update', 'mc-infra-manager', 'storage', 'update', 'Update Storage', 'Allow updating Storage via MCMP API'),
    ('mc-infra-manager:storage:delete', 'mc-infra-manager', 'storage', 'delete', 'Delete Storage', 'Allow deleting Storage via MCMP API'),
    ('mc-monitoring:monitoring:read', 'mc-monitoring', 'monitoring', 'read', 'Read Monitoring', 'Allow viewing monitoring data'),
    ('mc-monitoring:monitoring:update', 'mc-monitoring', 'monitoring', 'update', 'Update Monitoring', 'Allow updating monitoring settings')
ON CONFLICT (id) DO NOTHING;

-- Seed initial workspace roles (Example)
INSERT INTO mcmp_workspace_roles (name, description) VALUES
    ('admin', 'Workspace Administrator'),
    ('operator', 'Workspace Operator'),
    ('viewer', 'Workspace Viewer'),
    ('billadmin', 'Billing Administrator'),
    ('billviewer', 'Billing Viewer')
ON CONFLICT (name) DO NOTHING;

-- Assign permissions to roles (Example Mappings)
-- Clear existing workspace role permissions before seeding defaults
DELETE FROM mcmp_role_permissions WHERE role_type = 'workspace';

-- Workspace Admin Role
INSERT INTO mcmp_role_permissions (role_type, role_id, permission_id)
SELECT 'workspace', r.id, p.id
FROM mcmp_workspace_roles r, mcmp_permissions p
WHERE r.name = 'admin'
  AND p.framework_id IN ('mc-iam-manager', 'mc-infra-manager') -- Grant all IAM and Infra permissions
ON CONFLICT (role_type, role_id, permission_id) DO NOTHING;

-- Workspace Operator Role (Example: Infra RU + IAM Read)
INSERT INTO mcmp_role_permissions (role_type, role_id, permission_id)
SELECT 'workspace', r.id, p.id
FROM mcmp_workspace_roles r, mcmp_permissions p
WHERE r.name = 'operator'
  AND (
       (p.framework_id = 'mc-infra-manager' AND p.action IN ('read', 'update', 'delete')) OR
       (p.framework_id = 'mc-iam-manager' AND p.action = 'read')
      )
ON CONFLICT (role_type, role_id, permission_id) DO NOTHING;

-- Workspace Viewer Role (Example: Read only for IAM and Infra)
INSERT INTO mcmp_role_permissions (role_type, role_id, permission_id)
SELECT 'workspace', r.id, p.id
FROM mcmp_workspace_roles r, mcmp_permissions p
WHERE r.name = 'viewer'
  AND p.action = 'read'
  AND p.framework_id IN ('mc-iam-manager', 'mc-infra-manager')
ON CONFLICT (role_type, role_id, permission_id) DO NOTHING;

-- Billing Admin Role
INSERT INTO mcmp_role_permissions (role_type, role_id, permission_id)
SELECT 'workspace', r.id, p.id
FROM mcmp_workspace_roles r, mcmp_permissions p
WHERE r.name = 'billadmin'
  AND p.id IN ('mc-billing:billing:read', 'mc-billing:billing:update')
ON CONFLICT (role_type, role_id, permission_id) DO NOTHING;

-- Billing Viewer Role
INSERT INTO mcmp_role_permissions (role_type, role_id, permission_id)
SELECT 'workspace', r.id, p.id
FROM mcmp_workspace_roles r, mcmp_permissions p
WHERE r.name = 'billviewer'
  AND p.id = 'mc-billing:billing:read'
ON CONFLICT (role_type, workspace_role_id, permission_id) DO NOTHING;

-- Seed Role-CSP Role Mappings (Example - Add actual mappings as needed)
-- INSERT INTO mcmp_workspace_role_csp_role_mapping (workspace_role_id, csp_type, csp_role_arn, idp_identifier, description)
-- SELECT r.id, 'aws', 'arn:aws:iam::ACCOUNT:role/AdminRole', 'arn:aws:iam::ACCOUNT:oidc-provider/PROVIDER', 'Allow assuming AWS Admin Role'
-- FROM mcmp_workspace_roles r
-- WHERE r.name = 'admin'
-- ON CONFLICT (workspace_role_id, csp_type, csp_role_arn) DO NOTHING;


-- Note: Platform role permissions remain as they were
-- Assign all permissions to platformadmin (This might need adjustment based on new permission IDs)
-- INSERT INTO mcmp_role_permissions (role_type, role_id, permission_id)
-- SELECT 'platform', r.id, p.id
-- FROM mcmp_platform_roles r, mcmp_permissions p
-- WHERE r.name = 'platformadmin'
-- ON CONFLICT (role_type, role_id, permission_id) DO NOTHING;

-- Assign specific permissions to admin (This might need adjustment based on new permission IDs)
-- INSERT INTO mcmp_role_permissions (role_type, role_id, permission_id)
-- SELECT 'platform', r.id, p.id
-- FROM mcmp_platform_roles r, mcmp_permissions p
-- WHERE r.name = 'admin'
--   AND p.id IN ( ... list relevant new permission IDs ... )
-- ON CONFLICT (role_type, role_id, permission_id) DO NOTHING;
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
