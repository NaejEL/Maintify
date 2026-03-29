-- Migration 004: Add Time-Based Access Control
-- This migration adds support for scheduled role assignments and time-based access control

-- Scheduled role assignments table - For future role assignments with time-based activation
CREATE TABLE scheduled_role_assignments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    resource_scope LTREE, -- Optional: limit role to specific resource subtree
    scheduled_activation TIMESTAMP WITH TIME ZONE NOT NULL,
    scheduled_expiration TIMESTAMP WITH TIME ZONE, -- Optional: automatic expiration
    assigned_by UUID REFERENCES users(id),
    assignment_reason TEXT NOT NULL,
    notification_sent BOOLEAN DEFAULT false,
    is_processed BOOLEAN DEFAULT false,
    processed_at TIMESTAMP WITH TIME ZONE,
    processing_error TEXT,
    recurrence_pattern TEXT, -- CRON-like pattern for recurring access (e.g., "0 9 * * 1-5" for weekdays 9am)
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for performance
CREATE INDEX idx_scheduled_role_assignments_user_org ON scheduled_role_assignments(user_id, organization_id);
CREATE INDEX idx_scheduled_role_assignments_activation ON scheduled_role_assignments(scheduled_activation) WHERE NOT is_processed;
CREATE INDEX idx_scheduled_role_assignments_expiration ON scheduled_role_assignments(scheduled_expiration) WHERE is_processed AND scheduled_expiration IS NOT NULL;
CREATE INDEX idx_scheduled_role_assignments_pending ON scheduled_role_assignments(is_processed, scheduled_activation);
CREATE INDEX idx_scheduled_role_assignments_recurrence ON scheduled_role_assignments(recurrence_pattern) WHERE recurrence_pattern IS NOT NULL;

-- Add trigger for updated_at
CREATE TRIGGER update_scheduled_role_assignments_updated_at 
    BEFORE UPDATE ON scheduled_role_assignments 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Add metadata column to user_role_assignments for enhanced time-based control
ALTER TABLE user_role_assignments ADD COLUMN IF NOT EXISTS metadata JSONB DEFAULT '{}';

-- Add index for time-based queries on user_role_assignments
CREATE INDEX IF NOT EXISTS idx_user_role_assignments_time_based ON user_role_assignments(
    is_active, 
    valid_from, 
    valid_until
) WHERE valid_until IS NOT NULL;

-- Add function to clean up expired assignments automatically
CREATE OR REPLACE FUNCTION cleanup_expired_assignments()
RETURNS INTEGER AS $$
DECLARE
    expired_count INTEGER;
BEGIN
    -- Deactivate expired user role assignments
    UPDATE user_role_assignments 
    SET is_active = false, 
        updated_at = CURRENT_TIMESTAMP
    WHERE is_active = true 
      AND valid_until IS NOT NULL 
      AND valid_until < CURRENT_TIMESTAMP;
    
    GET DIAGNOSTICS expired_count = ROW_COUNT;
    
    -- Log cleanup action
    INSERT INTO rbac_audit (
        organization_id, 
        action, 
        resource_type, 
        success, 
        reason, 
        metadata
    ) 
    SELECT DISTINCT 
        organization_id,
        'cleanup_expired_assignments',
        'user_role_assignment',
        true,
        'Automatic cleanup of expired role assignments',
        jsonb_build_object('expired_count', expired_count)
    FROM user_role_assignments 
    WHERE is_active = false 
      AND valid_until IS NOT NULL 
      AND valid_until < CURRENT_TIMESTAMP
      AND updated_at >= CURRENT_TIMESTAMP - INTERVAL '1 minute';
    
    RETURN expired_count;
END;
$$ LANGUAGE plpgsql;

-- Add function to process scheduled activations
CREATE OR REPLACE FUNCTION process_scheduled_activations()
RETURNS INTEGER AS $$
DECLARE
    activation_count INTEGER := 0;
    scheduled_assignment RECORD;
    assignment_id UUID;
BEGIN
    -- Process all pending activations
    FOR scheduled_assignment IN 
        SELECT * FROM scheduled_role_assignments 
        WHERE NOT is_processed 
          AND scheduled_activation <= CURRENT_TIMESTAMP
        ORDER BY scheduled_activation ASC
    LOOP
        BEGIN
            -- Create the actual user role assignment
            INSERT INTO user_role_assignments (
                user_id,
                role_id,
                organization_id,
                resource_scope,
                valid_from,
                valid_until,
                assigned_by,
                assignment_reason,
                metadata
            ) VALUES (
                scheduled_assignment.user_id,
                scheduled_assignment.role_id,
                scheduled_assignment.organization_id,
                scheduled_assignment.resource_scope,
                scheduled_assignment.scheduled_activation,
                scheduled_assignment.scheduled_expiration,
                scheduled_assignment.assigned_by,
                scheduled_assignment.assignment_reason,
                scheduled_assignment.metadata || 
                jsonb_build_object(
                    'source', 'scheduled_assignment',
                    'scheduled_assignment_id', scheduled_assignment.id
                )
            ) RETURNING id INTO assignment_id;
            
            -- Mark scheduled assignment as processed
            UPDATE scheduled_role_assignments 
            SET is_processed = true,
                processed_at = CURRENT_TIMESTAMP,
                processing_error = NULL,
                updated_at = CURRENT_TIMESTAMP
            WHERE id = scheduled_assignment.id;
            
            -- Log successful activation
            INSERT INTO rbac_audit (
                organization_id,
                user_id,
                action,
                resource_type,
                resource_id,
                success,
                reason,
                metadata
            ) VALUES (
                scheduled_assignment.organization_id,
                scheduled_assignment.user_id,
                'activate_scheduled_role',
                'user_role_assignment',
                assignment_id,
                true,
                'Scheduled role assignment activated',
                jsonb_build_object(
                    'scheduled_assignment_id', scheduled_assignment.id,
                    'role_id', scheduled_assignment.role_id,
                    'activation_time', scheduled_assignment.scheduled_activation
                )
            );
            
            activation_count := activation_count + 1;
            
        EXCEPTION WHEN OTHERS THEN
            -- Log processing error
            UPDATE scheduled_role_assignments 
            SET processing_error = SQLERRM,
                updated_at = CURRENT_TIMESTAMP
            WHERE id = scheduled_assignment.id;
            
            INSERT INTO rbac_audit (
                organization_id,
                user_id,
                action,
                resource_type,
                success,
                reason,
                metadata
            ) VALUES (
                scheduled_assignment.organization_id,
                scheduled_assignment.user_id,
                'activate_scheduled_role',
                'scheduled_role_assignment',
                false,
                'Failed to activate scheduled role assignment',
                jsonb_build_object(
                    'scheduled_assignment_id', scheduled_assignment.id,
                    'error', SQLERRM
                )
            );
        END;
    END LOOP;
    
    RETURN activation_count;
END;
$$ LANGUAGE plpgsql;

-- Add some sample time-based permissions for common patterns
INSERT INTO permissions (organization_id, name, description, action) 
SELECT org.id, perm.name, perm.description, perm.action
FROM organizations org
CROSS JOIN (
    VALUES 
    ('schedule.manage', 'Manage scheduled access and time-based permissions', 'manage'),
    ('schedule.view', 'View scheduled access and time-based permissions', 'read'),
    ('emergency.override', 'Override time-based restrictions in emergencies', 'execute'),
    ('temporal.admin', 'Administer all time-based access controls', 'manage')
) AS perm(name, description, action)
WHERE org.slug = 'system';

-- Grant temporal admin permissions to system admin role
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'system-admin' 
  AND r.organization_id = (SELECT id FROM organizations WHERE slug = 'system')
  AND p.name IN ('schedule.manage', 'schedule.view', 'emergency.override', 'temporal.admin')
  AND p.organization_id = (SELECT id FROM organizations WHERE slug = 'system')
  AND NOT EXISTS (
    SELECT 1 FROM role_permissions rp 
    WHERE rp.role_id = r.id AND rp.permission_id = p.id
  );