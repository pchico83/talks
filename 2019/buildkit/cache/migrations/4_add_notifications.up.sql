/* Not needed anymore, leaving it for reference
CREATE OR REPLACE FUNCTION notify_deployment_event() RETURNS TRIGGER AS $$
    DECLARE 
        notification json;
    
    BEGIN
        IF (TG_OP = 'DELETE') THEN
            notification = json_build_object(
                          'table',TG_TABLE_NAME,
                          'action', TG_OP,
                          'id', OLD.id,
                          'project', OLD.project);
        ELSE
            notification = json_build_object(
                          'table',TG_TABLE_NAME,
                          'action', TG_OP,
                          'id', NEW.id,
                          'project', NEW.project);
        END IF;

        PERFORM pg_notify('events', notification::text);
        RETURN NULL; 
    END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION notify_project_event() RETURNS TRIGGER AS $$
    DECLARE 
        notification json;
    
    BEGIN
        IF (TG_OP = 'DELETE') THEN
            notification = json_build_object(
                          'table',TG_TABLE_NAME,
                          'action', TG_OP,
                          'id', OLD.id);
        ELSE
            notification = json_build_object(
                          'table',TG_TABLE_NAME,
                          'action', TG_OP,
                          'id', NEW.id);
        END IF;

        PERFORM pg_notify('events', notification::text);
        RETURN NULL; 
    END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER notify_deployment_event
AFTER INSERT OR UPDATE OR DELETE ON deployments
    FOR EACH ROW EXECUTE PROCEDURE notify_deployment_event();

CREATE TRIGGER notify_project_event
AFTER INSERT OR UPDATE OR DELETE ON projects
    FOR EACH ROW EXECUTE PROCEDURE notify_project_event();
*/