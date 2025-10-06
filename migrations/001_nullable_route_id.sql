-- Migration: Make route_id nullable in request_logs table
-- This allows logging requests that don't match any route in the database

-- Remove existing foreign key constraint
ALTER TABLE request_logs DROP CONSTRAINT IF EXISTS request_logs_route_id_fkey;

-- Make route_id nullable
ALTER TABLE request_logs ALTER COLUMN route_id DROP NOT NULL;

-- Add back foreign key constraint with ON DELETE SET NULL
ALTER TABLE request_logs 
ADD CONSTRAINT request_logs_route_id_fkey 
FOREIGN KEY (route_id) REFERENCES routes(id) ON DELETE SET NULL;

-- Add comment explaining nullable route_id
COMMENT ON COLUMN request_logs.route_id IS 'Route ID - nullable for requests that do not match any configured route';
