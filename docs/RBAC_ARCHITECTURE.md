# Enterprise RBAC Architecture

## Overview

Maintify's Role-Based Access Control (RBAC) system is designed to handle the full spectrum from small single-building operations to large multi-tenant enterprise maintenance companies. The architecture emphasizes flexibility, security, and user-friendly administration.

## Core Design Principles

### 1. **Maximum Flexibility**
- Dynamic resource hierarchies adaptable to any organizational structure
- Custom role creation without developer intervention
- Granular permission system with resource-level access control
- Time-based and conditional access patterns

### 2. **Enterprise Security**
- Multi-tenant data isolation
- Emergency privilege escalation with audit trails
- Comprehensive logging of all authentication and authorization events
- Conditional access (time-based, emergency, contractor schedules)

### 3. **User-Friendly Administration**
- HR-friendly role management interfaces
- Role templates for quick setup
- Non-developer custom role creation
- Visual hierarchy management

## Database Schema

### Organizations (Multi-Tenancy)
```sql
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    type organization_type NOT NULL DEFAULT 'self_hosted',
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TYPE organization_type AS ENUM ('self_hosted', 'saas_client');
```

### Dynamic Resource System
```sql
-- Resource types (Building, Room, Equipment, Region, Client, etc.)
CREATE TABLE resource_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(50) NOT NULL,
    parent_types UUID[] DEFAULT '{}',  -- Allowed parent resource types
    metadata JSONB DEFAULT '{}',       -- Custom fields definition
    icon VARCHAR(50),                  -- UI icon reference
    color VARCHAR(7),                  -- UI color hex code
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT unique_org_resource_type UNIQUE(organization_id, slug)
);

-- Actual resources (Building A, Room 101, HVAC Unit #5)
CREATE TABLE resources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    resource_type_id UUID NOT NULL REFERENCES resource_types(id) ON DELETE CASCADE,
    parent_id UUID REFERENCES resources(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL,
    description TEXT,
    metadata JSONB DEFAULT '{}',       -- Custom properties
    path LTREE,                        -- Hierarchical path for efficient queries
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT unique_org_resource UNIQUE(organization_id, resource_type_id, slug)
);

CREATE INDEX idx_resources_path ON resources USING GIST (path);
CREATE INDEX idx_resources_org_type ON resources(organization_id, resource_type_id);
```

### Users & Authentication
```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(100),
    password_hash VARCHAR(255),
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    employee_id VARCHAR(50),
    department VARCHAR(100),
    manager_id UUID REFERENCES users(id) ON DELETE SET NULL,
    is_active BOOLEAN DEFAULT true,
    is_locked BOOLEAN DEFAULT false,
    failed_login_attempts INTEGER DEFAULT 0,
    last_login TIMESTAMP WITH TIME ZONE,
    password_changed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT unique_org_email UNIQUE(organization_id, email)
);

CREATE INDEX idx_users_org_active ON users(organization_id, is_active);
```

### Permission System
```sql
-- System-wide available actions
CREATE TABLE actions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    category VARCHAR(50),
    is_system_action BOOLEAN DEFAULT true
);

-- Insert standard actions
INSERT INTO actions (name, description, category) VALUES 
    ('read', 'View/read access', 'basic'),
    ('write', 'Create/update access', 'basic'),
    ('delete', 'Delete access', 'basic'),
    ('admin', 'Administrative access', 'admin'),
    ('approve', 'Approval permissions', 'workflow'),
    ('execute', 'Execute operations', 'workflow'),
    ('emergency', 'Emergency access', 'emergency'),
    ('audit', 'Audit trail access', 'audit');

-- Organization-specific permissions
CREATE TABLE permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    action_id UUID NOT NULL REFERENCES actions(id),
    resource_type_id UUID REFERENCES resource_types(id) ON DELETE CASCADE,  -- NULL = system-wide
    conditions JSONB DEFAULT '{}',     -- Time-based, emergency, etc.
    is_system_permission BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT unique_org_permission UNIQUE(organization_id, name)
);
```

### Role System with Templates
```sql
-- Role templates (reusable across organizations)
CREATE TABLE role_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    category VARCHAR(50),
    default_permissions JSONB DEFAULT '[]',  -- Array of permission configurations
    metadata JSONB DEFAULT '{}',
    is_system_template BOOLEAN DEFAULT false,
    created_by_org UUID REFERENCES organizations(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT unique_template_name UNIQUE(name, created_by_org)
);

-- Organization roles
CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    parent_role_id UUID REFERENCES roles(id) ON DELETE SET NULL,  -- Role inheritance
    template_id UUID REFERENCES role_templates(id) ON DELETE SET NULL,
    is_custom BOOLEAN DEFAULT true,
    color VARCHAR(7),                  -- UI color
    icon VARCHAR(50),                  -- UI icon
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT unique_org_role UNIQUE(organization_id, name)
);

-- Role permissions mapping
CREATE TABLE role_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    granted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    granted_by UUID REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT unique_role_permission UNIQUE(role_id, permission_id)
);
```

### Advanced Assignment System
```sql
-- Time-based and conditional user-role assignments
CREATE TABLE user_role_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    resource_id UUID REFERENCES resources(id) ON DELETE CASCADE,  -- NULL = organization-wide
    
    -- Time-based access
    starts_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE,
    
    -- Conditional access
    conditions JSONB DEFAULT '{}',     -- Time patterns, emergency conditions, etc.
    
    -- Assignment metadata
    assigned_by UUID REFERENCES users(id) ON DELETE SET NULL,
    assigned_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    assignment_reason TEXT,
    is_active BOOLEAN DEFAULT true,
    
    -- Contractor/temporary access
    is_temporary BOOLEAN DEFAULT false,
    max_concurrent_sessions INTEGER DEFAULT 1,
    
    CONSTRAINT unique_user_role_resource UNIQUE(user_id, role_id, resource_id)
);

CREATE INDEX idx_assignments_user_active ON user_role_assignments(user_id, is_active);
CREATE INDEX idx_assignments_expires ON user_role_assignments(expires_at) WHERE expires_at IS NOT NULL;
```

### Emergency & Privilege Escalation
```sql
-- Emergency access logs and temporary privilege escalation
CREATE TABLE emergency_access (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    elevated_role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    resource_id UUID REFERENCES resources(id) ON DELETE CASCADE,
    
    -- Emergency details
    reason TEXT NOT NULL,
    emergency_type VARCHAR(50),        -- 'security', 'maintenance', 'safety', etc.
    approved_by UUID REFERENCES users(id) ON DELETE SET NULL,
    auto_approved BOOLEAN DEFAULT false,
    
    -- Time constraints
    activated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    deactivated_at TIMESTAMP WITH TIME ZONE,
    
    -- Audit trail
    actions_taken JSONB DEFAULT '[]',  -- Log of actions performed with elevated access
    metadata JSONB DEFAULT '{}',
    
    CONSTRAINT valid_emergency_timespan CHECK (expires_at > activated_at)
);

CREATE INDEX idx_emergency_user_active ON emergency_access(user_id) WHERE deactivated_at IS NULL;
```

### Audit & Logging Integration
```sql
-- Enhanced audit trail for RBAC operations
CREATE TABLE rbac_audit (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL,      -- 'role_assigned', 'permission_granted', 'emergency_activated'
    target_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    target_role_id UUID REFERENCES roles(id) ON DELETE SET NULL,
    target_resource_id UUID REFERENCES resources(id) ON DELETE SET NULL,
    old_values JSONB,
    new_values JSONB,
    ip_address INET,
    user_agent TEXT,
    session_id VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_rbac_audit_org_time ON rbac_audit(organization_id, created_at);
CREATE INDEX idx_rbac_audit_user ON rbac_audit(user_id);
```

## Key Features

### 1. **Flexible Resource Hierarchies**
- Dynamic resource types per organization
- LTREE for efficient hierarchical queries
- Support for multiple parent types (matrix organization structures)

### 2. **Time-Based Access Control**
- Contractor schedules (specific hours/days/weeks)
- Shift-based access (day/night teams)
- Temporary assignments with automatic expiration

### 3. **Emergency Privilege Escalation**
- Temporary elevated access with approval workflows
- Comprehensive audit trails
- Time-limited with automatic revocation

### 4. **Role Templates & Inheritance**
- System-provided templates for common roles
- Organization-specific custom templates
- Role inheritance for hierarchical permission structures

### 5. **Multi-Tenant Security**
- Complete data isolation per organization
- Resource-scoped permissions
- Cross-tenant access prevention

## API Design Patterns

### Permission Check Function
```go
type AccessContext struct {
    UserID       string
    Action       string
    ResourceID   *string
    ResourceType *string
    Emergency    bool
    Conditions   map[string]interface{}
}

func (rbac *RBACService) HasAccess(ctx AccessContext) (bool, error) {
    // 1. Check direct role assignments
    // 2. Check inherited permissions from parent resources
    // 3. Evaluate time-based conditions
    // 4. Check emergency access if applicable
    // 5. Log access attempt for audit
}
```

### Resource Hierarchy Queries
```sql
-- Get all resources user can access with specific action
WITH RECURSIVE user_accessible_resources AS (
    -- Direct assignments
    SELECT r.id, r.name, r.path, ura.role_id
    FROM resources r
    JOIN user_role_assignments ura ON ura.resource_id = r.id
    WHERE ura.user_id = $1 AND ura.is_active = true
    
    UNION
    
    -- Inherited from parent resources
    SELECT r.id, r.name, r.path, ura.role_id
    FROM resources r
    JOIN user_role_assignments ura ON r.path ~ (SELECT path FROM resources WHERE id = ura.resource_id)
    WHERE ura.user_id = $1 AND ura.is_active = true
)
SELECT DISTINCT uar.id, uar.name
FROM user_accessible_resources uar
JOIN role_permissions rp ON rp.role_id = uar.role_id
JOIN permissions p ON p.id = rp.permission_id
JOIN actions a ON a.id = p.action_id
WHERE a.name = $2;
```

## Implementation Plan

### Phase 1: Core Foundation
1. Database schema implementation
2. Basic RBAC service with organization isolation
3. User authentication and session management

### Phase 2: Resource & Permission Management
1. Dynamic resource type creation
2. Permission system with action mapping
3. Basic role assignment functionality

### Phase 3: Advanced Features
1. Time-based access control
2. Emergency privilege escalation
3. Role templates and inheritance

### Phase 4: Integration
1. Plugin RBAC integration
2. Audit logging with TimescaleDB
3. Admin UI for role management

## Management Interface

The RBAC system includes a comprehensive web-based management interface accessible through the Maintify Dashboard. This interface allows administrators to manage the entire access control lifecycle without direct database access.

### Features

#### 1. User Management
- **User List**: View all users in the organization with their status and roles.
- **User Creation**: Create new users with automatic organization scoping.
- **Role Assignment**: Assign roles to users directly from the user list.
- **Deactivation**: Soft-delete/deactivate users to revoke access immediately.

#### 2. Role Management
- **Role List**: View system and custom roles.
- **Role Creation**: Create custom roles specific to the organization.
- **Permission Assignment**: Granularly assign or remove permissions from roles via a visual interface.
- **System Roles**: View built-in system roles (read-only to prevent accidental lockout).

#### 3. Permission Management
- **Permission Registry**: View all available system permissions.
- **Custom Permissions**: Define new permissions (e.g., `report.view`, `equipment.maintain`) with specific actions (`create`, `read`, `update`, `delete`, `manage`).

#### 4. Audit Logs
- **Activity Timeline**: View a chronological history of security events.
- **Detailed Events**: Inspect specific actions, including who performed them, the target resource, and the outcome (success/failure).
- **Filtering**: (Planned) Filter logs by user, action, or date range.

## Third-Party Plugin Integration

### API Access for External Developers

The RBAC system is designed to be fully accessible to third-party plugin developers through a comprehensive REST API, **without requiring access to Maintify core code or internal services**. Plugins interact with RBAC through HTTP API calls to the core service.

#### Plugin Authentication & Authorization

1. **Plugin User Accounts**: Each plugin gets its own service account user with specific permissions
2. **JWT Token Authentication**: Plugins authenticate once and receive a JWT token for subsequent API calls
3. **Scoped Permissions**: Plugin users are assigned roles with permissions limited to their required functionality
4. **Organization Context**: All plugin operations are automatically scoped to the user's organization

#### Available RBAC API Endpoints for Plugins

**Authentication:**
```
POST /api/rbac/auth/login
- Authenticate plugin user and receive JWT token
- Body: {"email": "plugin@org.com", "password": "secure-pass"}
- Response: {"token": "jwt-token-here"}
```

**Permission Checking:**
```
GET /api/rbac/permissions/check?action=read&resource_type=equipment&resource_id=eq-123
- Verify if current user can perform action on resource
- Headers: Authorization: Bearer {jwt-token}
- Response: {"has_permission": true}
```

**User Information:**
```
GET /api/rbac/user/current
- Get authenticated user details and organization context
- Response: User object with organization_id, roles, etc.

GET /api/rbac/user/permissions
- Get all permissions for current user
- Response: Array of permission objects
```

**Resource Access:**
```
GET /api/rbac/organizations/{orgId}/resources?type=equipment
- List all resources user has access to
- Automatically filtered by user's permissions

GET /api/rbac/organizations/{orgId}/resources/{resourceId}
- Get specific resource details if user has access
```

**Audit Logging:**
```
POST /api/rbac/organizations/{orgId}/audit
- Create audit log entries for plugin actions
- Body: {"action": "maintenance_scheduled", "description": "...", "metadata": {}}
```

**Emergency Access:**
```
POST /api/rbac/organizations/{orgId}/emergency-access
- Request temporary elevated permissions
- Body: {"role_id": "uuid", "reason": "emergency maintenance", "duration": "2h"}
```

#### Plugin Development Workflow

1. **Service Account Setup**: Maintify administrator creates a dedicated user account for the plugin
2. **Role Assignment**: Plugin user is assigned appropriate roles (e.g., "Equipment Reader", "Maintenance Scheduler")
3. **API Integration**: Plugin uses HTTP client to authenticate and make RBAC calls
4. **Permission Checks**: Before any sensitive operation, plugin verifies user permissions
5. **Audit Trail**: Plugin logs all significant actions through RBAC audit API

#### Example Plugin Implementation

See `RBAC_PLUGIN_CLIENT_EXAMPLE.go` for a complete example of:
- Authenticating plugin users
- Checking permissions before operations
- Retrieving user and resource information
- Creating audit logs
- Requesting emergency access

#### Security Benefits for Plugin Developers

- **No Direct Database Access**: Plugins never need database credentials or internal system access
- **Automatic Multi-tenancy**: All operations are automatically scoped to the correct organization
- **Comprehensive Auditing**: All plugin actions are logged for security and compliance
- **Fine-grained Permissions**: Plugins can implement role-based features using Maintify's permission system
- **Emergency Procedures**: Support for break-glass access during critical situations

#### Integration Examples

**Equipment Management Plugin:**
```go
// Check if user can modify equipment before allowing changes
canModify, err := rbacClient.CheckPermission("write", "equipment", equipmentID)
if !canModify {
    return errors.New("insufficient permissions to modify equipment")
}

// Log the maintenance action
rbacClient.CreateAuditLog("equipment_modified", 
    fmt.Sprintf("Equipment %s updated by plugin", equipmentID),
    map[string]interface{}{
        "equipment_id": equipmentID,
        "plugin_name": "equipment-manager",
        "changes": changeList,
    })
```

**Work Order Plugin:**
```go
// Get all buildings user has access to
buildings, err := rbacClient.ListResources("building")

// Create work orders only for accessible locations
for _, building := range buildings {
    canCreateWorkOrder, _ := rbacClient.CheckPermission("write", "work_order", building.ID)
    if canCreateWorkOrder {
        // Create work order for this building
    }
}
```

This design ensures that third-party developers can build sophisticated, multi-tenant, permission-aware plugins without needing access to Maintify's internal architecture or database.