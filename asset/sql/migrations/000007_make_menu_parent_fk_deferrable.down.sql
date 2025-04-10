-- Revert the parent_id foreign key constraint in mcmp_menu to NOT DEFERRABLE

-- Step 1: Drop the deferrable foreign key constraint.
-- Use the same name used in the up migration (mcmp_menu_parent_id_fkey).
ALTER TABLE mcmp_menu DROP CONSTRAINT IF EXISTS mcmp_menu_parent_id_fkey;

-- Step 2: Re-add the foreign key constraint as NOT DEFERRABLE (default).
ALTER TABLE mcmp_menu ADD CONSTRAINT mcmp_menu_parent_id_fkey
    FOREIGN KEY (parent_id) REFERENCES mcmp_menu(id) ON DELETE CASCADE;
    -- NOT DEFERRABLE is the default and usually doesn't need to be specified explicitly.
