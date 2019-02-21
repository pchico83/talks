/* Not needed anymore, leaving it for reference
ALTER TABLE projects DROP COLUMN version;
ALTER TABLE users DROP COLUMN verified;
ALTER TABLE users DROP COLUMN created;
ALTER TABLE users DROP COLUMN updated;
ALTER TABLE users DROP COLUMN version;

DROP INDEX IF EXISTS ix_users_email;
DROP INDEX IF EXISTS ix_users_token;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_pkey;
ALTER TABLE users ADD PRIMARY KEY (id, token, email);
*/