-- ========================================
-- RBAC System Database Migration Script
-- ========================================
-- This script creates the necessary tables and seed data for the RBAC system
-- Run this AFTER creating the core master database tables

-- ========================================
-- 1. Create Master Database Tables
-- ========================================

-- Create access_roles table
CREATE TABLE IF NOT EXISTS "master".access_roles (
    id UUID PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    scope_level VARCHAR(50) NOT NULL, -- system, workspace, base
    priority INTEGER NOT NULL DEFAULT 0,
    description TEXT,
    is_default BOOLEAN NOT NULL DEFAULT false,
    created_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_modified_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_access_roles_name ON "master".access_roles(name);
CREATE INDEX IF NOT EXISTS idx_access_roles_scope_level ON "master".access_roles(scope_level);
CREATE INDEX IF NOT EXISTS idx_access_roles_priority ON "master".access_roles(priority);

-- Create resources table
CREATE TABLE IF NOT EXISTS "master".resources (
    id UUID PRIMARY KEY,
    code VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    created_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_resources_code ON "master".resources(code);

-- Create actions table
CREATE TABLE IF NOT EXISTS "master".actions (
    id UUID PRIMARY KEY,
    code VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    created_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_actions_code ON "master".actions(code);

-- Create permissions table (resource × action)
CREATE TABLE IF NOT EXISTS "master".permissions (
    id UUID PRIMARY KEY,
    resource_id UUID NOT NULL REFERENCES "master".resources(id) ON DELETE CASCADE,
    action_id UUID NOT NULL REFERENCES "master".actions(id) ON DELETE CASCADE,
    created_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(resource_id, action_id)
);

CREATE INDEX IF NOT EXISTS idx_permissions_resource_id ON "master".permissions(resource_id);
CREATE INDEX IF NOT EXISTS idx_permissions_action_id ON "master".permissions(action_id);

-- Create role_permissions table
CREATE TABLE IF NOT EXISTS "master".role_permissions (
    id UUID PRIMARY KEY,
    role_id UUID NOT NULL REFERENCES "master".access_roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES "master".permissions(id) ON DELETE CASCADE,
    created_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(role_id, permission_id)
);

CREATE INDEX IF NOT EXISTS idx_role_permissions_role_id ON "master".role_permissions(role_id);
CREATE INDEX IF NOT EXISTS idx_role_permissions_permission_id ON "master".role_permissions(permission_id);

-- ========================================
-- 2. Seed Default Resources
-- ========================================

INSERT INTO "master".resources (id, code, description) VALUES
    ('550e8400-e29b-41d4-a716-446655440001'::uuid, 'workspace', 'Workspace resource'),
    ('550e8400-e29b-41d4-a716-446655440002'::uuid, 'base', 'Base resource'),
    ('550e8400-e29b-41d4-a716-446655440003'::uuid, 'table', 'Table resource'),
    ('550e8400-e29b-41d4-a716-446655440004'::uuid, 'records', 'Records resource'),
    ('550e8400-e29b-41d4-a716-446655440005'::uuid, 'members', 'Members resource'),
    ('550e8400-e29b-41d4-a716-446655440006'::uuid, 'views', 'Views resource'),
    ('550e8400-e29b-41d4-a716-446655440007'::uuid, 'settings', 'Settings resource'),
    ('550e8400-e29b-41d4-a716-446655440008'::uuid, 'api_tokens', 'API Tokens resource'),
    ('550e8400-e29b-41d4-a716-446655440009'::uuid, 'webhooks', 'Webhooks resource'),
    ('550e8400-e29b-41d4-a716-446655440010'::uuid, 'automations', 'Automations resource')
ON CONFLICT DO NOTHING;

-- ========================================
-- 3. Seed Default Actions
-- ========================================

INSERT INTO "master".actions (id, code, description) VALUES
    ('660e8400-e29b-41d4-a716-446655440001'::uuid, 'read', 'Read access'),
    ('660e8400-e29b-41d4-a716-446655440002'::uuid, 'create', 'Create access'),
    ('660e8400-e29b-41d4-a716-446655440003'::uuid, 'update', 'Update access'),
    ('660e8400-e29b-41d4-a716-446655440004'::uuid, 'delete', 'Delete access'),
    ('660e8400-e29b-41d4-a716-446655440005'::uuid, 'share', 'Share access'),
    ('660e8400-e29b-41d4-a716-446655440006'::uuid, 'invite', 'Invite access'),
    ('660e8400-e29b-41d4-a716-446655440007'::uuid, 'export', 'Export access'),
    ('660e8400-e29b-41d4-a716-446655440008'::uuid, 'import', 'Import access'),
    ('660e8400-e29b-41d4-a716-446655440009'::uuid, 'execute', 'Execute access'),
    ('660e8400-e29b-41d4-a716-446655440010'::uuid, 'manage', 'Manage access')
ON CONFLICT DO NOTHING;

-- ========================================
-- 4. Seed Default Roles
-- ========================================

INSERT INTO "master".access_roles (id, name, scope_level, priority, description, is_default) VALUES
    ('770e8400-e29b-41d4-a716-446655440001'::uuid, 'owner', 'workspace', 100, 'Workspace owner with full control and management capabilities', false),
    ('770e8400-e29b-41d4-a716-446655440002'::uuid, 'co-owner', 'workspace', 90, 'Co-owner with full access similar to owner', false),
    ('770e8400-e29b-41d4-a716-446655440003'::uuid, 'workspace_maintainer', 'workspace', 80, 'Workspace maintainer with elevated permissions to manage workspace', false),
    ('770e8400-e29b-41d4-a716-446655440004'::uuid, 'workspace_maintainer_readonly', 'workspace', 70, 'Workspace maintainer with read-only access', false),
    ('770e8400-e29b-41d4-a716-446655440005'::uuid, 'base_member', 'base', 60, 'Base member with standard read and write permissions', false),
    ('770e8400-e29b-41d4-a716-446655440006'::uuid, 'base_member_readonly', 'base', 50, 'Base member with read-only access', false),
    ('770e8400-e29b-41d4-a716-446655440007'::uuid, 'viewer', 'base', 40, 'Viewer with minimal read-only access', false)
ON CONFLICT DO NOTHING;

-- ========================================
-- 5. Seed Permissions (Resource × Action)
-- ========================================

-- Workspace permissions
INSERT INTO "master".permissions (id, resource_id, action_id) VALUES
    ('880e8400-e29b-41d4-a716-446655440001'::uuid, '550e8400-e29b-41d4-a716-446655440001'::uuid, '660e8400-e29b-41d4-a716-446655440001'::uuid),
    ('880e8400-e29b-41d4-a716-446655440002'::uuid, '550e8400-e29b-41d4-a716-446655440001'::uuid, '660e8400-e29b-41d4-a716-446655440002'::uuid),
    ('880e8400-e29b-41d4-a716-446655440003'::uuid, '550e8400-e29b-41d4-a716-446655440001'::uuid, '660e8400-e29b-41d4-a716-446655440003'::uuid),
    ('880e8400-e29b-41d4-a716-446655440004'::uuid, '550e8400-e29b-41d4-a716-446655440001'::uuid, '660e8400-e29b-41d4-a716-446655440004'::uuid),
    ('880e8400-e29b-41d4-a716-446655440005'::uuid, '550e8400-e29b-41d4-a716-446655440001'::uuid, '660e8400-e29b-41d4-a716-446655440005'::uuid),
    ('880e8400-e29b-41d4-a716-446655440006'::uuid, '550e8400-e29b-41d4-a716-446655440001'::uuid, '660e8400-e29b-41d4-a716-446655440006'::uuid),

    -- Base permissions
    ('880e8400-e29b-41d4-a716-446655440010'::uuid, '550e8400-e29b-41d4-a716-446655440002'::uuid, '660e8400-e29b-41d4-a716-446655440001'::uuid),
    ('880e8400-e29b-41d4-a716-446655440011'::uuid, '550e8400-e29b-41d4-a716-446655440002'::uuid, '660e8400-e29b-41d4-a716-446655440002'::uuid),
    ('880e8400-e29b-41d4-a716-446655440012'::uuid, '550e8400-e29b-41d4-a716-446655440002'::uuid, '660e8400-e29b-41d4-a716-446655440003'::uuid),
    ('880e8400-e29b-41d4-a716-446655440013'::uuid, '550e8400-e29b-41d4-a716-446655440002'::uuid, '660e8400-e29b-41d4-a716-446655440004'::uuid),

    -- Records permissions
    ('880e8400-e29b-41d4-a716-446655440020'::uuid, '550e8400-e29b-41d4-a716-446655440004'::uuid, '660e8400-e29b-41d4-a716-446655440001'::uuid),
    ('880e8400-e29b-41d4-a716-446655440021'::uuid, '550e8400-e29b-41d4-a716-446655440004'::uuid, '660e8400-e29b-41d4-a716-446655440002'::uuid),
    ('880e8400-e29b-41d4-a716-446655440022'::uuid, '550e8400-e29b-41d4-a716-446655440004'::uuid, '660e8400-e29b-41d4-a716-446655440003'::uuid),
    ('880e8400-e29b-41d4-a716-446655440023'::uuid, '550e8400-e29b-41d4-a716-446655440004'::uuid, '660e8400-e29b-41d4-a716-446655440004'::uuid),
    ('880e8400-e29b-41d4-a716-446655440024'::uuid, '550e8400-e29b-41d4-a716-446655440004'::uuid, '660e8400-e29b-41d4-a716-446655440007'::uuid),

    -- Members permissions
    ('880e8400-e29b-41d4-a716-446655440030'::uuid, '550e8400-e29b-41d4-a716-446655440005'::uuid, '660e8400-e29b-41d4-a716-446655440001'::uuid),
    ('880e8400-e29b-41d4-a716-446655440031'::uuid, '550e8400-e29b-41d4-a716-446655440005'::uuid, '660e8400-e29b-41d4-a716-446655440006'::uuid),
    ('880e8400-e29b-41d4-a716-446655440032'::uuid, '550e8400-e29b-41d4-a716-446655440005'::uuid, '660e8400-e29b-41d4-a716-446655440010'::uuid)
ON CONFLICT DO NOTHING;

-- ========================================
-- 6. Assign Permissions to Owner Role
-- ========================================

INSERT INTO "master".role_permissions (id, role_id, permission_id) 
SELECT gen_random_uuid(), '770e8400-e29b-41d4-a716-446655440001'::uuid, id 
FROM "master".permissions 
WHERE id IN (
    '880e8400-e29b-41d4-a716-446655440001'::uuid,
    '880e8400-e29b-41d4-a716-446655440002'::uuid,
    '880e8400-e29b-41d4-a716-446655440003'::uuid,
    '880e8400-e29b-41d4-a716-446655440004'::uuid,
    '880e8400-e29b-41d4-a716-446655440005'::uuid,
    '880e8400-e29b-41d4-a716-446655440006'::uuid,
    '880e8400-e29b-41d4-a716-446655440010'::uuid,
    '880e8400-e29b-41d4-a716-446655440011'::uuid,
    '880e8400-e29b-41d4-a716-446655440012'::uuid,
    '880e8400-e29b-41d4-a716-446655440013'::uuid,
    '880e8400-e29b-41d4-a716-446655440030'::uuid,
    '880e8400-e29b-41d4-a716-446655440031'::uuid,
    '880e8400-e29b-41d4-a716-446655440032'::uuid
)
ON CONFLICT DO NOTHING;

-- ========================================
-- 7. Assign Permissions to Base Member Role
-- ========================================

INSERT INTO "master".role_permissions (id, role_id, permission_id) 
SELECT gen_random_uuid(), '770e8400-e29b-41d4-a716-446655440005'::uuid, id 
FROM "master".permissions 
WHERE id IN (
    '880e8400-e29b-41d4-a716-446655440010'::uuid,
    '880e8400-e29b-41d4-a716-446655440020'::uuid,
    '880e8400-e29b-41d4-a716-446655440021'::uuid,
    '880e8400-e29b-41d4-a716-446655440022'::uuid,
    '880e8400-e29b-41d4-a716-446655440023'::uuid
)
ON CONFLICT DO NOTHING;

-- ========================================
-- 8. Assign Permissions to Viewer Role
-- ========================================

INSERT INTO "master".role_permissions (id, role_id, permission_id) 
SELECT gen_random_uuid(), '770e8400-e29b-41d4-a716-446655440007'::uuid, id 
FROM "master".permissions 
WHERE id IN (
    '880e8400-e29b-41d4-a716-446655440010'::uuid,
    '880e8400-e29b-41d4-a716-446655440020'::uuid
)
ON CONFLICT DO NOTHING;

-- ========================================
-- Verification Queries
-- ========================================

-- Verify roles created
-- SELECT * FROM "master".access_roles;

-- Verify resources created
-- SELECT * FROM "master".resources;

-- Verify actions created
-- SELECT * FROM "master".actions;

-- Verify permissions created
-- SELECT * FROM "master".permissions;

-- Verify role permissions created
-- SELECT * FROM "master".role_permissions;
