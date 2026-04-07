-- Migrate any rows with legacy role values to 'platform_member' before applying
-- the new constraint. 'member' and 'admin' are the only prior values in use;
-- both map to 'platform_member' under the OAuth-only model. No silent data
-- corruption occurs: the UPDATE is explicit and auditable.
UPDATE users SET role = 'platform_member' WHERE role IN ('member', 'admin');

-- Drop the old check constraint and replace it with the two-value constraint.
ALTER TABLE users DROP CONSTRAINT users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('platform_admin', 'platform_member'));

-- Set the column default to 'platform_member' for all new rows.
ALTER TABLE users ALTER COLUMN role SET DEFAULT 'platform_member';
