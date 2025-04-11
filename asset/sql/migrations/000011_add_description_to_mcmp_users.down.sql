-- Remove description column from mcmp_users table
ALTER TABLE mcmp_users
DROP COLUMN IF EXISTS description;
