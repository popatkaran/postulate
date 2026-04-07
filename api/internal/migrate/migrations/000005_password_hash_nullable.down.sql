-- WARNING: POTENTIALLY DESTRUCTIVE.
-- Restoring NOT NULL will fail if any row has a NULL password_hash.
-- Before running this down migration, verify that no OAuth-only users exist,
-- or back-fill password_hash with a sentinel value first.
ALTER TABLE users ALTER COLUMN password_hash SET NOT NULL;
