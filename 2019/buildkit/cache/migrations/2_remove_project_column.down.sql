ALTER TABLE users ADD COLUMN IF NOT EXISTS project text;
ALTER TABLE projects ADD COLUMN IF NOT EXISTS type text;