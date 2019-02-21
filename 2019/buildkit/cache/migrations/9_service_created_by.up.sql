ALTER TABLE services ADD COLUMN IF NOT EXISTS created_by text;
ALTER TABLE services ADD COLUMN IF NOT EXISTS is_demo boolean;

CREATE INDEX IF NOT EXISTS service_created_by ON services(created_by);
CREATE INDEX IF NOT EXISTS service_is_demo ON services(is_demo);
