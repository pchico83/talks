ALTER TABLE projectacls DROP COLUMN IF EXISTS role;
ALTER TABLE deployments DROP COLUMN IF EXISTS dns;
ALTER TABLE projects DROP COLUMN IF EXISTS dnsname;

DROP INDEX IF EXISTS deployment_unique_name;
DROP INDEX IF EXISTS project_unique_dns;