ALTER TABLE services DROP COLUMN IF EXISTS created_by;
ALTER TABLE services DROP COLUMN IF EXISTS is_demo;

DROP INDEX IF EXISTS service_created_by;
DROP INDEX IF EXISTS service_is_demo;