-- Create mcmp_workspaces table
CREATE TABLE IF NOT EXISTS mcmp_workspaces (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create mcmp_projects table
CREATE TABLE IF NOT EXISTS mcmp_projects (
    id SERIAL PRIMARY KEY,
    nsid VARCHAR(255), -- Consider adding UNIQUE constraint if needed globally or per workspace (latter needs composite key)
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create mcmp_workspace_projects mapping table for M:N relationship
CREATE TABLE IF NOT EXISTS mcmp_workspace_projects (
    workspace_id INTEGER NOT NULL REFERENCES mcmp_workspaces(id) ON DELETE CASCADE,
    project_id INTEGER NOT NULL REFERENCES mcmp_projects(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (workspace_id, project_id)
);
