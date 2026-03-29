-- Migration 005: Enhanced Emergency Access with Break-Glass Procedures and Approval Workflows
-- This migration adds sophisticated emergency access request workflows, approval mechanisms,
-- and break-glass procedures for critical maintenance scenarios.

-- Emergency access request statuses
CREATE TYPE emergency_access_request_status AS ENUM (
    'pending',
    'approved', 
    'denied',
    'expired',
    'revoked',
    'granted'
);

-- Emergency urgency levels
CREATE TYPE emergency_urgency_level AS ENUM (
    'low',
    'medium', 
    'high',
    'critical'
);

-- Emergency access approval actions
CREATE TYPE emergency_access_approval_action AS ENUM (
    'approve',
    'deny'
);

-- Emergency access requests table
CREATE TABLE emergency_access_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    requested_permissions TEXT[] NOT NULL DEFAULT '{}',
    reason TEXT NOT NULL,
    urgency_level emergency_urgency_level NOT NULL DEFAULT 'medium',
    requested_duration BIGINT NOT NULL DEFAULT 7200000000000, -- 2 hours in nanoseconds
    break_glass BOOLEAN NOT NULL DEFAULT false,
    required_approvals INTEGER NOT NULL DEFAULT 1,
    status emergency_access_request_status NOT NULL DEFAULT 'pending',
    requested_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMPTZ,
    auto_approved_at TIMESTAMPTZ,
    emergency_access_id UUID REFERENCES emergency_access(id) ON DELETE SET NULL,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Emergency access approvals table
CREATE TABLE emergency_access_approvals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id UUID NOT NULL REFERENCES emergency_access_requests(id) ON DELETE CASCADE,
    approver_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    action emergency_access_approval_action NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Break-glass configuration table
CREATE TABLE break_glass_config (
    organization_id UUID PRIMARY KEY REFERENCES organizations(id) ON DELETE CASCADE,
    enabled BOOLEAN NOT NULL DEFAULT false,
    auto_approval_urgency emergency_urgency_level NOT NULL DEFAULT 'critical',
    max_duration INTERVAL NOT NULL DEFAULT '4 hours',
    required_permissions TEXT[] NOT NULL DEFAULT '{}',
    approval_requirements JSONB NOT NULL DEFAULT '{"low": 2, "medium": 2, "high": 1, "critical": 0}',
    auto_revocation_minutes INTEGER NOT NULL DEFAULT 240,
    notification_channels TEXT[] NOT NULL DEFAULT '{}',
    escalation_rules JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_emergency_access_requests_user_org ON emergency_access_requests(user_id, organization_id);
CREATE INDEX idx_emergency_access_requests_status ON emergency_access_requests(status);
CREATE INDEX idx_emergency_access_requests_urgency ON emergency_access_requests(urgency_level);
CREATE INDEX idx_emergency_access_requests_expires_at ON emergency_access_requests(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_emergency_access_requests_created_at ON emergency_access_requests(created_at);
CREATE INDEX idx_emergency_access_requests_break_glass ON emergency_access_requests(break_glass) WHERE break_glass = true;

CREATE INDEX idx_emergency_access_approvals_request ON emergency_access_approvals(request_id);
CREATE INDEX idx_emergency_access_approvals_approver ON emergency_access_approvals(approver_id);
CREATE INDEX idx_emergency_access_approvals_created_at ON emergency_access_approvals(created_at);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_emergency_access_requests_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Trigger for updated_at on emergency_access_requests
CREATE TRIGGER update_emergency_access_requests_updated_at_trigger
    BEFORE UPDATE ON emergency_access_requests
    FOR EACH ROW
    EXECUTE FUNCTION update_emergency_access_requests_updated_at();

-- Trigger for updated_at on break_glass_config
CREATE TRIGGER update_break_glass_config_updated_at_trigger
    BEFORE UPDATE ON break_glass_config
    FOR EACH ROW
    EXECUTE FUNCTION update_emergency_access_requests_updated_at();

-- Function to automatically expire emergency access requests
CREATE OR REPLACE FUNCTION expire_emergency_access_requests()
RETURNS void AS $$
BEGIN
    UPDATE emergency_access_requests 
    SET status = 'expired'
    WHERE status = 'pending' 
      AND expires_at IS NOT NULL 
      AND expires_at < CURRENT_TIMESTAMP;
END;
$$ language 'plpgsql';

-- Function to process break-glass access
CREATE OR REPLACE FUNCTION process_break_glass_access(request_id_param UUID)
RETURNS void AS $$
DECLARE
    req_record emergency_access_requests%ROWTYPE;
    config_record break_glass_config%ROWTYPE;
    access_id UUID;
BEGIN
    -- Get the request
    SELECT * INTO req_record 
    FROM emergency_access_requests 
    WHERE id = request_id_param AND status = 'pending';
    
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Emergency access request not found or not pending';
    END IF;
    
    -- Get break-glass config
    SELECT * INTO config_record 
    FROM break_glass_config 
    WHERE organization_id = req_record.organization_id;
    
    IF NOT FOUND OR NOT config_record.enabled THEN
        RAISE EXCEPTION 'Break-glass access not enabled for organization';
    END IF;
    
    -- Check if auto-approval criteria are met
    IF req_record.urgency_level::text >= config_record.auto_approval_urgency::text THEN
        -- Auto-approve and grant access
        UPDATE emergency_access_requests 
        SET status = 'approved', auto_approved_at = CURRENT_TIMESTAMP
        WHERE id = request_id_param;
        
        -- Create emergency access grant
        INSERT INTO emergency_access (
            user_id, organization_id, granted_permissions, reason,
            granted_by, approved_by, valid_from, valid_until, is_active
        ) VALUES (
            req_record.user_id,
            req_record.organization_id,
            req_record.requested_permissions,
            'BREAK-GLASS: ' || req_record.reason,
            req_record.user_id, -- Self-granted in break-glass
            req_record.user_id, -- Self-approved in break-glass
            CURRENT_TIMESTAMP,
            CURRENT_TIMESTAMP + LEAST(req_record.requested_duration, config_record.max_duration),
            true
        ) RETURNING id INTO access_id;
        
        -- Link the access to the request
        UPDATE emergency_access_requests 
        SET emergency_access_id = access_id, status = 'granted'
        WHERE id = request_id_param;
    END IF;
END;
$$ language 'plpgsql';

-- Function to check approval requirements and grant access
CREATE OR REPLACE FUNCTION check_approval_requirements(request_id_param UUID)
RETURNS void AS $$
DECLARE
    req_record emergency_access_requests%ROWTYPE;
    config_record break_glass_config%ROWTYPE;
    approval_count INTEGER;
    required_approvals INTEGER;
    access_id UUID;
BEGIN
    -- Get the request
    SELECT * INTO req_record 
    FROM emergency_access_requests 
    WHERE id = request_id_param AND status = 'pending';
    
    IF NOT FOUND THEN
        RETURN;
    END IF;
    
    -- Count approvals
    SELECT COUNT(*) INTO approval_count
    FROM emergency_access_approvals
    WHERE request_id = request_id_param AND action = 'approve';
    
    -- Get required approvals
    SELECT * INTO config_record 
    FROM break_glass_config 
    WHERE organization_id = req_record.organization_id;
    
    IF FOUND THEN
        required_approvals := (config_record.approval_requirements->req_record.urgency_level::text)::INTEGER;
    ELSE
        required_approvals := req_record.required_approvals;
    END IF;
    
    -- Check if we have enough approvals
    IF approval_count >= required_approvals THEN
        -- Update request status
        UPDATE emergency_access_requests 
        SET status = 'approved'
        WHERE id = request_id_param;
        
        -- Create emergency access grant
        INSERT INTO emergency_access (
            user_id, organization_id, granted_permissions, reason,
            granted_by, valid_from, valid_until, is_active
        ) VALUES (
            req_record.user_id,
            req_record.organization_id,
            req_record.requested_permissions,
            req_record.reason,
            (SELECT approver_id FROM emergency_access_approvals 
             WHERE request_id = request_id_param AND action = 'approve' 
             ORDER BY created_at DESC LIMIT 1),
            CURRENT_TIMESTAMP,
            CURRENT_TIMESTAMP + req_record.requested_duration,
            true
        ) RETURNING id INTO access_id;
        
        -- Link the access to the request
        UPDATE emergency_access_requests 
        SET emergency_access_id = access_id, status = 'granted'
        WHERE id = request_id_param;
    END IF;
END;
$$ language 'plpgsql';

-- Trigger to check approval requirements after each approval
CREATE OR REPLACE FUNCTION trigger_check_approval_requirements()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.action = 'approve' THEN
        PERFORM check_approval_requirements(NEW.request_id);
    ELSIF NEW.action = 'deny' THEN
        UPDATE emergency_access_requests 
        SET status = 'denied'
        WHERE id = NEW.request_id AND status = 'pending';
    END IF;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER emergency_access_approval_trigger
    AFTER INSERT ON emergency_access_approvals
    FOR EACH ROW
    EXECUTE FUNCTION trigger_check_approval_requirements();

-- Insert default break-glass configurations for existing organizations
INSERT INTO break_glass_config (organization_id, enabled)
SELECT id, false 
FROM organizations 
WHERE NOT EXISTS (
    SELECT 1 FROM break_glass_config WHERE organization_id = organizations.id
);

COMMENT ON TABLE emergency_access_requests IS 'Emergency access requests with approval workflows';
COMMENT ON TABLE emergency_access_approvals IS 'Approvals and denials for emergency access requests';
COMMENT ON TABLE break_glass_config IS 'Break-glass configuration per organization';
COMMENT ON FUNCTION process_break_glass_access(UUID) IS 'Process break-glass access for critical scenarios';
COMMENT ON FUNCTION check_approval_requirements(UUID) IS 'Check if approval requirements are met and grant access';
COMMENT ON FUNCTION expire_emergency_access_requests() IS 'Automatically expire pending emergency access requests';