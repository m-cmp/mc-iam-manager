-- Initial data seeding based on migrations 000005, 000009, 000010

-- Seed initial platform roles (from 000010)
INSERT INTO mcmp_platform_roles (name, description) VALUES
    ('platformadmin', 'Platform Administrator with all privileges'),
    ('admin', 'Administrator role with user management capabilities'),
    ('operator', 'Operator role with operational privileges'),
    ('viewer', 'Viewer role with read-only access'),
    ('billadmin', 'Billing administrator role'),
    ('billviewer', 'Billing viewer role')
ON CONFLICT (name) DO NOTHING;

-- Seed initial menu data (from 000005)
INSERT INTO mcmp_menu (id, parent_id, display_name, res_type, is_action, priority, menu_number)
VALUES ('dashboard', NULL, 'Dashboard', 'menu', false, 1, 1)
ON CONFLICT (id) DO NOTHING;
-- Note: Add other essential menus if they were previously seeded via YAML or other migrations

-- Seed default workspace
INSERT INTO mcmp_workspaces (name, description) VALUES
    ('ws01', 'Default workspace for unassigned projects')
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
DELETE FROM mcmp_mciam_permissions; -- Clear old permissions first (Updated table name)

INSERT INTO mcmp_mciam_permissions (id, framework_id, resource_type_id, action, name, description) VALUES -- Updated table name
    -- IAM Workspace Permissions
    ('mc-iam-manager:workspace:create', 'mc-iam-manager', 'workspace', 'create', 'Create Workspace', 'Allow creating workspaces'),
    ('mc-iam-manager:workspace:read', 'mc-iam-manager', 'workspace', 'read', 'Read Workspace', 'Allow viewing workspaces'),
    ('mc-iam-manager:workspace:update', 'mc-iam-manager', 'workspace', 'update', 'Update Workspace', 'Allow updating workspaces'),
    ('mc-iam-manager:workspace:delete', 'mc-iam-manager', 'workspace', 'delete', 'Delete Workspace', 'Allow deleting workspaces'),
    ('mc-iam-manager:workspace:assign_user', 'mc-iam-manager', 'workspace', 'assign_user', 'Assign User to Workspace', 'Allow assigning users to workspaces'),
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
    ('mc-billing:billing:update', 'mc-billing', 'billing', 'update', 'Update Billing', 'Allow updating billing settings')
    -- Add other permissions for k8s, vpc, storage, monitoring etc.
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
DELETE FROM mcmp_mciam_role_permissions WHERE role_type = 'workspace'; -- Updated table name

-- Workspace Admin Role
INSERT INTO mcmp_mciam_role_permissions (role_type, workspace_role_id, permission_id) -- Updated table name and column name
SELECT 'workspace', r.id, p.id
FROM mcmp_workspace_roles r, mcmp_mciam_permissions p -- Updated table name
WHERE r.name = 'admin'
  AND p.framework_id IN ('mc-iam-manager', 'mc-infra-manager') -- Grant all IAM and Infra permissions
ON CONFLICT (role_type, workspace_role_id, permission_id) DO NOTHING; -- Updated conflict target

-- Workspace Operator Role (Example: Infra RU + IAM Read)
INSERT INTO mcmp_mciam_role_permissions (role_type, workspace_role_id, permission_id) -- Updated table name and column name
SELECT 'workspace', r.id, p.id
FROM mcmp_workspace_roles r, mcmp_mciam_permissions p -- Updated table name
WHERE r.name = 'operator'
  AND (
       (p.framework_id = 'mc-infra-manager' AND p.action IN ('read', 'update', 'delete')) OR
       (p.framework_id = 'mc-iam-manager' AND p.action = 'read')
      )
ON CONFLICT (role_type, workspace_role_id, permission_id) DO NOTHING; -- Updated conflict target

-- Workspace Viewer Role (Example: Read only for IAM and Infra)
INSERT INTO mcmp_mciam_role_permissions (role_type, workspace_role_id, permission_id) -- Updated table name and column name
SELECT 'workspace', r.id, p.id
FROM mcmp_workspace_roles r, mcmp_mciam_permissions p -- Updated table name
WHERE r.name = 'viewer'
  AND p.action = 'read'
  AND p.framework_id IN ('mc-iam-manager', 'mc-infra-manager')
ON CONFLICT (role_type, workspace_role_id, permission_id) DO NOTHING; -- Updated conflict target

-- Billing Admin Role
INSERT INTO mcmp_mciam_role_permissions (role_type, workspace_role_id, permission_id) -- Updated table name and column name
SELECT 'workspace', r.id, p.id
FROM mcmp_workspace_roles r, mcmp_mciam_permissions p -- Updated table name
WHERE r.name = 'billadmin'
  AND p.id IN ('mc-billing:billing:read', 'mc-billing:billing:update')
ON CONFLICT (role_type, workspace_role_id, permission_id) DO NOTHING; -- Updated conflict target

-- Billing Viewer Role
INSERT INTO mcmp_mciam_role_permissions (role_type, workspace_role_id, permission_id) -- Updated table name and column name
SELECT 'workspace', r.id, p.id
FROM mcmp_workspace_roles r, mcmp_mciam_permissions p -- Updated table name
WHERE r.name = 'billviewer'
  AND p.id = 'mc-billing:billing:read'
ON CONFLICT (role_type, workspace_role_id, permission_id) DO NOTHING;

-- Seed Role-CSP Role Mappings (Example - Add actual mappings as needed)
-- INSERT INTO mcmp_workspace_role_csp_role_mapping (workspace_role_id, csp_type, csp_role_arn, idp_identifier, description)
-- SELECT r.id, 'aws', 'arn:aws:iam::ACCOUNT_ID:role/MCMP_admin', 'arn:aws:iam::ACCOUNT_ID:oidc-provider/KEYCLOAK_HOSTNAME', 'Mapping for Workspace Admin to AWS MCMP_admin')
-- FROM mcmp_workspace_roles r
-- WHERE r.name = 'admin'
-- ON CONFLICT (workspace_role_id, csp_type, csp_role_arn) DO NOTHING;
-- INSERT INTO mcmp_workspace_role_csp_role_mapping (workspace_role_id, csp_type, csp_role_arn, idp_identifier, description)
-- SELECT r.id, 'aws', 'arn:aws:iam::ACCOUNT_ID:role/MCMP_viewer', 'arn:aws:iam::ACCOUNT_ID:oidc-provider/KEYCLOAK_HOSTNAME', 'Mapping for Workspace Viewer to AWS MCMP_viewer')
-- FROM mcmp_workspace_roles r
-- WHERE r.name = 'viewer' -- Assuming viewer role ID is 3 based on previous context
-- ON CONFLICT (workspace_role_id, csp_type, csp_role_arn) DO NOTHING;


-- Note: Platform role permissions remain as they were
-- Assign all permissions to platformadmin (This might need adjustment based on new permission IDs)
-- INSERT INTO mcmp_mciam_role_permissions (role_type, workspace_role_id, permission_id) -- Assuming platform roles use this table and workspace_role_id column
-- SELECT 'platform', r.id, p.id
-- FROM mcmp_platform_roles r, mcmp_mciam_permissions p
-- WHERE r.name = 'platformadmin'
-- ON CONFLICT (role_type, workspace_role_id, permission_id) DO NOTHING; -- Updated conflict target

-- Assign specific permissions to admin (This might need adjustment based on new permission IDs)
-- INSERT INTO mcmp_mciam_role_permissions (role_type, workspace_role_id, permission_id) -- Assuming platform roles use this table
-- SELECT 'platform', r.id, p.id
-- FROM mcmp_platform_roles r, mcmp_mciam_permissions p
-- WHERE r.name = 'admin'
--   AND p.id IN ( ... list relevant new permission IDs ... )
-- ON CONFLICT (role_type, workspace_role_id, permission_id) DO NOTHING; -- Updated conflict target
    ('accountnaccess', 'accountnaccess', 'Allow viewing account & access menu'),
    ('organizations', 'organizations', 'Allow viewing organizations menu'),
    ('users', 'users', 'Allow viewing users menu'),
    ('operations', 'operations', 'Allow viewing operations menu'),
    ('manage', 'manage', 'Allow viewing manage menu'),
    ('workspaces', 'workspaces', 'Allow viewing workspaces menu'),
    -- User Management Permissions
    ('user:create', 'user:create', 'Allow creating new users (admin only)'),
    ('user:list', 'user:list', 'Allow listing users'),
    ('user:get', 'user:get', 'Allow getting user details'),
    ('user:update', 'user:update', 'Allow updating user details'),
    ('user:delete', 'user:delete', 'Allow deleting users'),
    ('user:approve', 'user:approve', 'Allow approving user registration requests'),
    ('user:reject', 'user:reject', 'Allow rejecting user registration requests'),
    ('registration:list', 'registration:list', 'Allow listing pending registration requests') -- Keep or remove based on final logic
    -- Add other necessary permissions...
ON CONFLICT (id) DO NOTHING;

-- Assign permissions to roles (from 000009 and 000010) - Platform Roles
-- Assign all permissions to platformadmin
INSERT INTO mcmp_mciam_role_permissions (role_type, workspace_role_id, permission_id) -- Updated table/column names
SELECT 'platform', r.id, p.id
FROM mcmp_platform_roles r, mcmp_mciam_permissions p
WHERE r.name = 'platformadmin'
ON CONFLICT (role_type, workspace_role_id, permission_id) DO NOTHING;

-- Assign specific permissions to admin
INSERT INTO mcmp_mciam_role_permissions (role_type, workspace_role_id, permission_id) -- Updated table/column names
SELECT 'platform', r.id, p.id
FROM mcmp_platform_roles r, mcmp_mciam_permissions p
WHERE r.name = 'admin'
  AND p.id IN (
    -- Menu permissions for admin (example, adjust as needed)
    'dashboard', 'settings', 'accountnaccess', 'organizations', 'users', 'operations', 'manage', 'workspaces',
    -- User management permissions
    'user:create', 'user:list', 'user:get', 'user:update', 'user:delete', 'user:approve', 'user:reject', 'registration:list'
    -- Add other admin-specific permission IDs...
  )
ON CONFLICT (role_type, workspace_role_id, permission_id) DO NOTHING; -- Updated conflict target

-- Assign permissions for other roles (example for 'viewer')
-- INSERT INTO mcmp_mciam_role_permissions (role_type, workspace_role_id, permission_id) -- Updated table/column names
-- SELECT 'platform', r.id, p.id
-- FROM mcmp_platform_roles r, mcmp_mciam_permissions p
-- WHERE r.name = 'viewer'
--   AND p.id IN ('dashboard') -- Only dashboard view permission
-- ON CONFLICT (role_type, workspace_role_id, permission_id) DO NOTHING; -- Updated conflict target

-- 권한-API 액션 매핑 초기 데이터
-- 먼저 기존 매핑 데이터 삭제
DELETE FROM mcmp_mciam_permission_action_mappings;

-- MC-IAM Manager 권한 매핑
INSERT INTO mcmp_mciam_permission_action_mappings (permission_id, action_id)
SELECT 
    p.id as permission_id,
    a.id as action_id
FROM mcmp_mciam_permissions p
JOIN mcmp_api_actions a ON 
    -- 워크스페이스 관련 권한 매핑
    (p.id = 'mc-iam-manager:workspace:create' AND a.action_name = 'CreateWorkspace') OR
    (p.id = 'mc-iam-manager:workspace:read' AND a.action_name = 'GetWorkspace') OR
    (p.id = 'mc-iam-manager:workspace:update' AND a.action_name = 'UpdateWorkspace') OR
    (p.id = 'mc-iam-manager:workspace:delete' AND a.action_name = 'DeleteWorkspace') OR
    (p.id = 'mc-iam-manager:workspace:assign_user' AND a.action_name = 'AssignUserToWorkspace') OR
    -- 프로젝트 관련 권한 매핑
    (p.id = 'mc-iam-manager:project:create' AND a.action_name = 'CreateProject') OR
    (p.id = 'mc-iam-manager:project:read' AND a.action_name = 'GetProject') OR
    (p.id = 'mc-iam-manager:project:update' AND a.action_name = 'UpdateProject') OR
    (p.id = 'mc-iam-manager:project:delete' AND a.action_name = 'DeleteProject') OR
    (p.id = 'mc-iam-manager:project:sync' AND a.action_name = 'SyncProjects')
ON CONFLICT (permission_id, action_id) DO NOTHING;

-- MC-Infra Manager 권한 매핑
INSERT INTO mcmp_mciam_permission_action_mappings (permission_id, action_id)
SELECT 
    p.id as permission_id,
    a.id as action_id
FROM mcmp_mciam_permissions p
JOIN mcmp_api_actions a ON 
    -- VM 관련 권한 매핑
    (p.id = 'mc-infra-manager:vm:create' AND a.action_name = 'CreateVm') OR
    (p.id = 'mc-infra-manager:vm:read' AND a.action_name = 'GetVm') OR
    (p.id = 'mc-infra-manager:vm:update' AND a.action_name = 'UpdateVm') OR
    (p.id = 'mc-infra-manager:vm:delete' AND a.action_name = 'DeleteVm') OR
    -- K8s 관련 권한 매핑
    (p.id = 'mc-infra-manager:k8s:create' AND a.action_name = 'CreateK8sCluster') OR
    (p.id = 'mc-infra-manager:k8s:read' AND a.action_name = 'GetK8sCluster') OR
    (p.id = 'mc-infra-manager:k8s:update' AND a.action_name = 'UpdateK8sCluster') OR
    (p.id = 'mc-infra-manager:k8s:delete' AND a.action_name = 'DeleteK8sCluster') OR
    -- VPC 관련 권한 매핑
    (p.id = 'mc-infra-manager:vpc:create' AND a.action_name = 'CreateVpc') OR
    (p.id = 'mc-infra-manager:vpc:read' AND a.action_name = 'GetVpc') OR
    (p.id = 'mc-infra-manager:vpc:update' AND a.action_name = 'UpdateVpc') OR
    (p.id = 'mc-infra-manager:vpc:delete' AND a.action_name = 'DeleteVpc')
ON CONFLICT (permission_id, action_id) DO NOTHING;

-- MC-Billing 권한 매핑
INSERT INTO mcmp_mciam_permission_action_mappings (permission_id, action_id)
SELECT 
    p.id as permission_id,
    a.id as action_id
FROM mcmp_mciam_permissions p
JOIN mcmp_api_actions a ON 
    (p.id = 'mc-billing:billing:read' AND a.action_name = 'GetBillingInfo') OR
    (p.id = 'mc-billing:billing:update' AND a.action_name = 'UpdateBillingSettings')
ON CONFLICT (permission_id, action_id) DO NOTHING;
