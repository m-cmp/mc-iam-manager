-- IAM-FIX-001: mcmp_menus menu resource columns
-- See ST3_개발/MIGRATION-GUIDE-mcmp-menus-menu-resource.md

BEGIN;

ALTER TABLE mcmp_menus
  ADD COLUMN IF NOT EXISTS view_type VARCHAR(20) NOT NULL DEFAULT 'local',
  ADD COLUMN IF NOT EXISTS framework_service VARCHAR(100) NOT NULL DEFAULT 'mc-web-console-front',
  ADD COLUMN IF NOT EXISTS path VARCHAR(500) NOT NULL DEFAULT '';

COMMENT ON COLUMN mcmp_menus.view_type IS 'local | iframe | popup';
COMMENT ON COLUMN mcmp_menus.framework_service IS 'getapihosts service key';
COMMENT ON COLUMN mcmp_menus.path IS 'path within framework';

COMMIT;
