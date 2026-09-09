-- IAM-TECH-035: mcmp_menus icon column

BEGIN;

ALTER TABLE mcmp_menus
  ADD COLUMN IF NOT EXISTS icon VARCHAR(100) NOT NULL DEFAULT '';

COMMENT ON COLUMN mcmp_menus.icon IS 'icon identifier for menu item (frontend icon set key)';

COMMIT;
