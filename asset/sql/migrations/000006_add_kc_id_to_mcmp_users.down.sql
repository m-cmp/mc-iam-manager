-- Remove kc_id column and its unique constraint from mcmp_users table
ALTER TABLE mcmp_users
DROP CONSTRAINT IF EXISTS mcmp_users_kc_id_key;

ALTER TABLE mcmp_users
DROP COLUMN IF EXISTS kc_id;
