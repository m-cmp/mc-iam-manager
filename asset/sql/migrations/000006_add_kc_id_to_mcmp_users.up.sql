-- Add kc_id column to mcmp_users table to store Keycloak User ID
ALTER TABLE mcmp_users
ADD COLUMN kc_id VARCHAR(255);

-- Update existing rows with a placeholder or fetch from Keycloak if possible (difficult in pure SQL)
-- UPDATE mcmp_users SET kc_id = 'placeholder_kc_id' WHERE kc_id IS NULL;

-- Add NOT NULL constraint after potentially populating existing rows
-- ALTER TABLE mcmp_users ALTER COLUMN kc_id SET NOT NULL;

-- Add UNIQUE constraint
ALTER TABLE mcmp_users
ADD CONSTRAINT mcmp_users_kc_id_key UNIQUE (kc_id);

-- Note: Populating kc_id for existing users might require application logic
-- or manual updates depending on how Keycloak IDs map to existing DB users.
-- The NOT NULL constraint should only be added after ensuring all rows have a value.
