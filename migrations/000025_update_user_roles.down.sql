-- Migration 025: Revert user role check constraint
-- Reverts to the previous state without 'editor' and 'runner'.
-- Note: This will fail if there are existing users with 'editor' or 'runner' roles.

ALTER TABLE users
  DROP CONSTRAINT IF EXISTS users_role_check;

ALTER TABLE users
  ADD CONSTRAINT users_role_check
  CHECK (role IN ('super_admin', 'admin', 'operator', 'viewer'));
