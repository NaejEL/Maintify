-- Migration 001: Create RBAC Schema
-- This migration creates the foundational RBAC tables for multi-tenant access control

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "ltree";

-- Organizations table - Top-level tenant isolation
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT organizations_slug_format CHECK (slug ~ '^[a-z0-9-]+$')
);

-- Resource types table - Define what kinds of resources can be managed
CREATE TABLE resource_types (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    hierarchy_enabled BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(organization_id, name)
);

-- Resources table - Actual resources that can be accessed (buildings, rooms, equipment, etc.)
CREATE TABLE resources (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    resource_type_id UUID NOT NULL REFERENCES resource_types(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    parent_path LTREE,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(organization_id, resource_type_id, name)
);

-- Index for hierarchical queries
CREATE INDEX idx_resources_parent_path_gist ON resources USING GIST (parent_path);
CREATE INDEX idx_resources_org_type ON resources(organization_id, resource_type_id);

-- Users table - Authentication and basic user info
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    is_active BOOLEAN DEFAULT true,
    is_system_admin BOOLEAN DEFAULT false,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_login_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT users_email_format CHECK (email ~ '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$')
);

-- Permissions table - Granular actions that can be performed
CREATE TABLE permissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    resource_type_id UUID REFERENCES resource_types(id) ON DELETE CASCADE,
    action VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(organization_id, name),
    CONSTRAINT permissions_action_valid CHECK (action IN ('create', 'read', 'update', 'delete', 'execute', 'manage'))
);

-- Roles table - Groups of permissions
CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    is_system_role BOOLEAN DEFAULT false,
    is_template BOOLEAN DEFAULT false,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(organization_id, name)
);

-- Role permissions junction table
CREATE TABLE role_permissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(role_id, permission_id)
);

-- User role assignments with optional time-based access and resource scope
CREATE TABLE user_role_assignments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    resource_scope LTREE, -- Optional: limit role to specific resource subtree
    valid_from TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    valid_until TIMESTAMP WITH TIME ZONE, -- Optional: time-limited access
    assigned_by UUID REFERENCES users(id),
    assignment_reason TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Emergency access table - Temporary privilege escalation
CREATE TABLE emergency_access (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    granted_permissions TEXT[] NOT NULL,
    reason TEXT NOT NULL,
    granted_by UUID REFERENCES users(id),
    approved_by UUID REFERENCES users(id),
    valid_from TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    valid_until TIMESTAMP WITH TIME ZONE NOT NULL,
    is_active BOOLEAN DEFAULT true,
    revoked_at TIMESTAMP WITH TIME ZONE,
    revoked_by UUID REFERENCES users(id),
    revoke_reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- RBAC audit log - Track all permission changes and access attempts
CREATE TABLE rbac_audit (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(50) NOT NULL,
    resource_type VARCHAR(100),
    resource_id UUID,
    permission_name VARCHAR(100),
    success BOOLEAN NOT NULL,
    reason TEXT,
    ip_address INET,
    user_agent TEXT,
    session_id VARCHAR(255),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for performance
CREATE INDEX idx_user_role_assignments_user_org ON user_role_assignments(user_id, organization_id);
CREATE INDEX idx_user_role_assignments_active ON user_role_assignments(is_active, valid_from, valid_until);
CREATE INDEX idx_emergency_access_user_org ON emergency_access(user_id, organization_id);
CREATE INDEX idx_emergency_access_active ON emergency_access(is_active, valid_from, valid_until);
CREATE INDEX idx_rbac_audit_org_user ON rbac_audit(organization_id, user_id, created_at);
CREATE INDEX idx_rbac_audit_resource ON rbac_audit(resource_type, resource_id, created_at);

-- Create unique constraint for user role assignments (handles nullable resource_scope)
CREATE UNIQUE INDEX idx_user_role_assignments_unique ON user_role_assignments(user_id, role_id, organization_id, COALESCE(resource_scope, ''));

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply updated_at triggers to relevant tables
CREATE TRIGGER update_organizations_updated_at BEFORE UPDATE ON organizations FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_resources_updated_at BEFORE UPDATE ON resources FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_roles_updated_at BEFORE UPDATE ON roles FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_user_role_assignments_updated_at BEFORE UPDATE ON user_role_assignments FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Insert default system organization
INSERT INTO organizations (name, slug, description, settings) VALUES 
('System', 'system', 'System organization for global settings and system users', '{"is_system": true}');

-- Insert basic resource types for the system organization
INSERT INTO resource_types (organization_id, name, description, hierarchy_enabled) VALUES 
((SELECT id FROM organizations WHERE slug = 'system'), 'system', 'System-wide resources', false),
((SELECT id FROM organizations WHERE slug = 'system'), 'building', 'Buildings and facilities', true),
((SELECT id FROM organizations WHERE slug = 'system'), 'room', 'Rooms within buildings', true),
((SELECT id FROM organizations WHERE slug = 'system'), 'equipment', 'Maintenance equipment and assets', true);

-- Insert basic system permissions
INSERT INTO permissions (organization_id, name, description, action) VALUES 
((SELECT id FROM organizations WHERE slug = 'system'), 'system.admin', 'Full system administration', 'manage'),
((SELECT id FROM organizations WHERE slug = 'system'), 'org.create', 'Create new organizations', 'create'),
((SELECT id FROM organizations WHERE slug = 'system'), 'org.manage', 'Manage organization settings', 'manage'),
((SELECT id FROM organizations WHERE slug = 'system'), 'user.manage', 'Manage user accounts', 'manage'),
((SELECT id FROM organizations WHERE slug = 'system'), 'rbac.manage', 'Manage roles and permissions', 'manage');

-- Insert system admin role
INSERT INTO roles (organization_id, name, description, is_system_role) VALUES 
((SELECT id FROM organizations WHERE slug = 'system'), 'system-admin', 'System administrator with full access', true);

-- Assign all system permissions to system admin role
INSERT INTO role_permissions (role_id, permission_id) 
SELECT 
    (SELECT id FROM roles WHERE name = 'system-admin' AND organization_id = (SELECT id FROM organizations WHERE slug = 'system')),
    id 
FROM permissions WHERE organization_id = (SELECT id FROM organizations WHERE slug = 'system');