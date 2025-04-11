-- Remove redundant columns from mcmp_users (already managed by Keycloak)
ALTER TABLE mcmp_users
DROP COLUMN IF EXISTS email,
DROP COLUMN IF EXISTS first_name,
DROP COLUMN IF EXISTS last_name;
