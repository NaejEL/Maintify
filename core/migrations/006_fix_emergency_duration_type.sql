-- Migration 006: Fix emergency access duration type
-- Convert requested_duration from INTERVAL to BIGINT (nanoseconds)

-- First drop the existing column constraints/functions that depend on it
DROP FUNCTION IF EXISTS process_break_glass_access(UUID);
DROP FUNCTION IF EXISTS check_approval_requirements(UUID);

-- Alter the column type by first dropping the default, then changing type, then setting new default
-- requested_duration is already BIGINT in 005, so we just ensure the default is correct
ALTER TABLE emergency_access_requests ALTER COLUMN requested_duration SET DEFAULT 7200000000000;

-- Update max_duration in break_glass_config as well
ALTER TABLE break_glass_config ALTER COLUMN max_duration DROP DEFAULT;
ALTER TABLE break_glass_config 
ALTER COLUMN max_duration TYPE BIGINT USING EXTRACT(EPOCH FROM max_duration) * 1000000000;
ALTER TABLE break_glass_config 
ALTER COLUMN max_duration SET DEFAULT 14400000000000;

-- Recreate the break-glass processing function with updated duration handling
CREATE OR REPLACE FUNCTION process_break_glass_access(request_id_param UUID)
RETURNS void AS $$
DECLARE
    req_record emergency_access_requests%ROWTYPE;
    config_record break_glass_config%ROWTYPE;
    access_id UUID;
    duration_seconds BIGINT;
    max_duration_seconds BIGINT;
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
    
    -- Convert nanoseconds to seconds for duration calculation
    duration_seconds := req_record.requested_duration / 1000000000;
    max_duration_seconds := config_record.max_duration / 1000000000;
    
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
            CURRENT_TIMESTAMP + INTERVAL '1 second' * LEAST(duration_seconds, max_duration_seconds),
            true
        ) RETURNING id INTO access_id;
        
        -- Link the access to the request
        UPDATE emergency_access_requests 
        SET emergency_access_id = access_id, status = 'granted'
        WHERE id = request_id_param;
    END IF;
END;
$$ language 'plpgsql';

-- Recreate the approval checking function with updated duration handling
CREATE OR REPLACE FUNCTION check_approval_requirements(request_id_param UUID)
RETURNS void AS $$
DECLARE
    req_record emergency_access_requests%ROWTYPE;
    config_record break_glass_config%ROWTYPE;
    approval_count INTEGER;
    required_approvals INTEGER;
    access_id UUID;
    duration_seconds BIGINT;
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
        
        -- Convert nanoseconds to seconds
        duration_seconds := req_record.requested_duration / 1000000000;
        
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
            CURRENT_TIMESTAMP + INTERVAL '1 second' * duration_seconds,
            true
        ) RETURNING id INTO access_id;
        
        -- Link the access to the request
        UPDATE emergency_access_requests 
        SET emergency_access_id = access_id, status = 'granted'
        WHERE id = request_id_param;
    END IF;
END;
$$ language 'plpgsql';

-- Fix emergency access duration type to use BIGINT (nanoseconds) instead of INTERVAL