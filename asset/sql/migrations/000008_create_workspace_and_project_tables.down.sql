-- Drop tables in reverse order of creation due to foreign key constraints
DROP TABLE IF EXISTS mcmp_workspace_projects;
DROP TABLE IF EXISTS mcmp_projects;
DROP TABLE IF EXISTS mcmp_workspaces;
