-- we catch exceptions because, in a new DB, we don't need any of this. only for staging/prod and similar
CREATE OR REPLACE FUNCTION migrate_projects() RETURNS VOID AS $$
BEGIN
    BEGIN
        ALTER TABLE projects ADD COLUMN IF NOT EXISTS dns_name text;
        UPDATE projects SET dns_name=dnsname;
        ALTER TABLE projects ALTER COLUMN  dns_name SET NOT NULL;
        ALTER TABLE projects ADD CONSTRAINT unique_dns_name UNIQUE (dns_name);
        ALTER TABLE projects DROP IF EXISTS dnsname;
    EXCEPTION WHEN OTHERS THEN
        NULL;
    END;

    BEGIN
        INSERT INTO 
            project_acls(project_id, "user_id", "role", created_at, updated_at)
            (SELECT project, "user", "role", now(), now() FROM projectacls);
    EXCEPTION WHEN OTHERS THEN
        NULL;
    END;
END;
$$ LANGUAGE plpgsql;

select migrate_projects();
DROP FUNCTION migrate_projects();