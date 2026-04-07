-- WARNING: POTENTIALLY DESTRUCTIVE.
-- Restoring NOT NULL will fail if any refresh_tokens row has a NULL session_id.
-- Verify no OAuth-issued tokens exist before rolling back.
ALTER TABLE refresh_tokens ALTER COLUMN session_id SET NOT NULL;
