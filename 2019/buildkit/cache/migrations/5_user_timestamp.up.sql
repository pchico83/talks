/* Not needed anymore, leaving it for reference
UPDATE users SET created_at=to_timestamp(created) WHERE created_at IS NULL;
ALTER TABLE users DROP COLUMN IF EXISTS created;

UPDATE users SET updated_at=to_timestamp(updated) WHERE updated_at IS NULL;
ALTER TABLE users DROP COLUMN IF EXISTS updated;

DROP TABLE IF EXISTS userwhitelist;
*/