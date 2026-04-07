-- Restore the prior check constraint and default.
-- NOTE: rows with 'platform_admin' or 'platform_member' that were inserted after
-- the up migration will not be automatically reverted. Verify data before rolling back.
ALTER TABLE users DROP CONSTRAINT users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('member', 'admin', 'platform_admin'));
ALTER TABLE users ALTER COLUMN role SET DEFAULT 'member';
