-- Purpose : Remove expired sessions. The application also does this hourly;
--           this script exists for manual maintenance.
-- Run as  : webapp_app or webapp_owner, connected to the application database.
-- Changes : Deletes rows from sessions whose expiry has passed.

SET search_path = webapp;
DELETE FROM sessions WHERE expires_at < now();
