-- Migration 025: Update user role check constraint
-- Includes 'editor' and 'runner' roles which were missing in previous constraint.

ALTER TABLE users
  DROP CONSTRAINT IF EXISTS users_role_check;

ALTER TABLE users
  ADD CONSTRAINT users_role_check
  CHECK (role IN ('super_admin', 'admin', 'editor', 'runner', 'operator', 'viewer'));
