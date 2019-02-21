DROP TRIGGER IF EXISTS project_notify_event ON projects;
DROP TRIGGER IF EXISTS deployment_notify_event ON deployments;
DROP FUNCTION IF EXISTS notify_event;