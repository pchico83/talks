/* Not needed anymore, leaving it for reference
ALTER TABLE projectacls ADD COLUMN IF NOT EXISTS role text;
UPDATE projectacls SET role='admin';

ALTER TABLE deployments ADD COLUMN IF NOT EXISTS dns text;
UPDATE deployments SET dns='';

ALTER TABLE projects ADD COLUMN IF NOT EXISTS dnsname text;
UPDATE projects SET dnsname="name";

CREATE UNIQUE INDEX IF NOT EXISTS deployment_unique_name ON deployments("name", project) WHERE (state != 'destroyed');
CREATE UNIQUE INDEX IF NOT EXISTS project_unique_dns ON projects(dnsname);
*/