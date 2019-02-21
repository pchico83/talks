CREATE OR REPLACE FUNCTION notify_service_event() RETURNS TRIGGER AS $$
    DECLARE 
        notification json;
    
    BEGIN
        IF (TG_OP = 'DELETE') THEN
            notification = json_build_object(
                          'table',TG_TABLE_NAME,
                          'action', TG_OP,
                          'id', OLD.id,
                          'project', OLD.project_id);
        ELSE
            notification = json_build_object(
                          'table',TG_TABLE_NAME,
                          'action', TG_OP,
                          'id', NEW.id,
                          'project', NEW.project_id);
        END IF;

        PERFORM pg_notify('events', notification::text);
        RETURN NULL; 
    END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER notify_service_event
  AFTER INSERT OR UPDATE OR DELETE ON services
    FOR EACH ROW EXECUTE PROCEDURE notify_service_event();


CREATE UNIQUE INDEX IF NOT EXISTS service_unique_name ON services("name", project_id) WHERE ("status" != 'destroyed');

-- we catch exceptions because, in a new DB, we don't need any of this. only for staging/prod and similar
CREATE OR REPLACE FUNCTION migrate_services() RETURNS VOID AS $$
BEGIN

    BEGIN
        INSERT INTO 
            services(id, "name", "status", project_id, dns, manifest, created_at, updated_at)
            (SELECT id, "name", "state", project, dns, manifest, now(), now() FROM deployments);
    EXCEPTION WHEN OTHERS THEN
        NULL;
    END;

    BEGIN
        INSERT INTO 
            activity_logs(id, activity_id, created_at,updated_at, "log") 
            (SELECT id, activity, to_timestamp(created), to_timestamp(created), "log" FROM activitylogs);
    EXCEPTION WHEN OTHERS THEN
        NULL;
    END;

    BEGIN
        UPDATE activities SET created_at=to_timestamp(created) WHERE created_at IS NULL;
        ALTER TABLE activities DROP COLUMN IF EXISTS created;
    EXCEPTION WHEN OTHERS THEN
        NULL;
    END;

    BEGIN
        UPDATE activities SET actor_id=actor WHERE actor_id IS NULL;
        ALTER TABLE activities DROP COLUMN IF EXISTS actor;
    EXCEPTION WHEN OTHERS THEN
        NULL;
    END;

    BEGIN
        UPDATE activities SET updated_at=to_timestamp(updated) WHERE updated_at IS NULL;
        ALTER TABLE activities DROP COLUMN IF EXISTS updated;
    EXCEPTION WHEN OTHERS THEN
        NULL;
    END;

    BEGIN
        UPDATE activities SET service_id=deployment WHERE service_id IS NULL;
        ALTER TABLE activities DROP COLUMN IF EXISTS deployment;
    EXCEPTION WHEN OTHERS THEN
        NULL;
    END;
END;
$$ LANGUAGE plpgsql;

select migrate_services();
DROP FUNCTION migrate_services();
UPDATE services SET dns=NULL WHERE status='destroyed';