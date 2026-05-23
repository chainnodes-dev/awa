-- Migration 020: Tenant Branding and Secrets (Down)
ALTER TABLE tenants DROP COLUMN IF EXISTS logo_url;
ALTER TABLE tenants DROP COLUMN IF EXISTS secrets;
