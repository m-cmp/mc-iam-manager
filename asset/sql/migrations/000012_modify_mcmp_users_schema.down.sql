-- Re-add columns removed in the up migration
ALTER TABLE mcmp_users
ADD COLUMN email VARCHAR(255),
ADD COLUMN first_name VARCHAR(255),
ADD COLUMN last_name VARCHAR(255);

-- Add back constraints if they were dropped (check original schema)
-- Example: ALTER TABLE mcmp_users ADD CONSTRAINT mcmp_users_email_key UNIQUE (email);
-- Example: ALTER TABLE mcmp_users ALTER COLUMN email SET NOT NULL;
-- Note: Populating the re-added columns with data might be necessary for NOT NULL constraints.
