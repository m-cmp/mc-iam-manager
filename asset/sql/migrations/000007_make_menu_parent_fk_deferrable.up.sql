-- Make the parent_id foreign key constraint in mcmp_menu deferrable

-- Step 1: Drop the existing foreign key constraint.
-- The actual constraint name might be different (e.g., mcmp_menu_parent_id_fkey, or something auto-generated).
-- Use \d mcmp_menu in psql to find the correct constraint name if the one below fails.
ALTER TABLE mcmp_menu DROP CONSTRAINT IF EXISTS mcmp_menu_parent_id_fkey;

-- Step 2: Re-add the foreign key constraint as DEFERRABLE INITIALLY DEFERRED.
-- Use the same constraint name or a new one.
ALTER TABLE mcmp_menu ADD CONSTRAINT mcmp_menu_parent_id_fkey
    FOREIGN KEY (parent_id) REFERENCES mcmp_menu(id) ON DELETE CASCADE
    DEFERRABLE INITIALLY DEFERRED;
