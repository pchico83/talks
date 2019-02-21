ALTER TABLE projects ADD COLUMN IF NOT EXISTS version bigint;
ALTER TABLE users ADD COLUMN IF NOT EXISTS verified boolean;
ALTER TABLE users ADD COLUMN IF NOT EXISTS created bigint;
ALTER TABLE users ADD COLUMN IF NOT EXISTS updated bigint;
ALTER TABLE users ADD COLUMN IF NOT EXISTS version bigint;
UPDATE users SET verified='t', version=1, created=1520239575, updated=1520239575;
UPDATE projects SET version=1;


CREATE UNIQUE INDEX IF NOT EXISTS ix_users_email ON users(email);
CREATE UNIQUE INDEX IF NOT EXISTS ix_users_token ON users(token);
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_pkey;
ALTER TABLE users ADD PRIMARY KEY (id);
