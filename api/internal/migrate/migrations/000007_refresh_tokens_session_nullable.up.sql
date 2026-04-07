-- Make session_id nullable so refresh tokens can be issued without a session row.
-- OAuth-issued tokens (US-03-04, US-03-05) do not create a sessions row;
-- session_id is NULL for these tokens. The FK constraint is retained for rows
-- that do reference a session.
ALTER TABLE refresh_tokens ALTER COLUMN session_id DROP NOT NULL;
