-- Migration 020: Tenant Branding and Secrets
-- Adds logo_url and secrets columns to the tenants table.

ALTER TABLE tenants ADD COLUMN logo_url TEXT;
ALTER TABLE tenants ADD COLUMN secrets JSONB NOT NULL DEFAULT '{}';
