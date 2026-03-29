-- TimescaleDB Logging Schema for Maintify
-- Designed for high-volume log storage with time-series optimization
-- Compatible with existing LogEntry structure from pkg/logger

-- Create TimescaleDB extension (must be done by superuser)
-- This will be handled in docker-compose setup
-- CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Create log levels enum for consistency
CREATE TYPE log_level AS ENUM ('DEBUG', 'INFO', 'WARN', 'ERROR', 'FATAL');

-- Main logs table optimized for time-series data
CREATE TABLE logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL,  -- Multi-tenant isolation (follows guidelines)
    
    -- Core log fields (matching existing LogEntry structure)
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    level log_level NOT NULL,
    message TEXT NOT NULL,
    component VARCHAR(100) NOT NULL,  -- "core", "builder", plugin names
    source VARCHAR(255),              -- File/line where log originated
    
    -- User context fields
    user_id UUID,                     -- References users.id from RBAC system
    session_id VARCHAR(255),          -- User session identifier
    request_id VARCHAR(255),          -- HTTP request correlation ID
    
    -- Plugin context fields
    plugin_name VARCHAR(100),         -- Plugin identifier for plugin logs
    action VARCHAR(100),              -- Action being performed
    
    -- Error and details
    error_message TEXT,               -- Error details if level is ERROR/FATAL
    details JSONB DEFAULT '{}',       -- Flexible metadata storage (follows guidelines)
    
    -- Search optimization
    message_vector TSVECTOR,          -- Full-text search on message content
    
    -- Audit trail
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Constraints for data integrity
    CONSTRAINT logs_org_check CHECK (organization_id IS NOT NULL),
    CONSTRAINT logs_timestamp_check CHECK (timestamp IS NOT NULL),
    CONSTRAINT logs_message_check CHECK (char_length(message) > 0)
);

-- Convert to TimescaleDB hypertable (time-series optimization)
-- Partition by timestamp with 1-day intervals for optimal performance
-- Note: Commented out for PostgreSQL-only setup. Uncomment when TimescaleDB is available.
-- SELECT create_hypertable('logs', 'timestamp', chunk_time_interval => INTERVAL '1 day');

-- Create indexes for high-performance queries following guidelines

-- Primary search indexes
CREATE INDEX idx_logs_org_timestamp ON logs (organization_id, timestamp DESC);
CREATE INDEX idx_logs_component_timestamp ON logs (component, timestamp DESC);
CREATE INDEX idx_logs_level_timestamp ON logs (level, timestamp DESC);
CREATE INDEX idx_logs_user_timestamp ON logs (user_id, timestamp DESC) WHERE user_id IS NOT NULL;

-- Plugin-specific indexes
CREATE INDEX idx_logs_plugin_timestamp ON logs (plugin_name, timestamp DESC) WHERE plugin_name IS NOT NULL;
CREATE INDEX idx_logs_action_timestamp ON logs (action, timestamp DESC) WHERE action IS NOT NULL;

-- Session and request correlation
CREATE INDEX idx_logs_session ON logs (session_id, timestamp DESC) WHERE session_id IS NOT NULL;
CREATE INDEX idx_logs_request ON logs (request_id, timestamp DESC) WHERE request_id IS NOT NULL;

-- Full-text search index
CREATE INDEX idx_logs_message_search ON logs USING GIN (message_vector);

-- JSONB details search (for complex metadata queries)
CREATE INDEX idx_logs_details ON logs USING GIN (details);

-- Composite indexes for common query patterns
CREATE INDEX idx_logs_org_component_level ON logs (organization_id, component, level, timestamp DESC);
CREATE INDEX idx_logs_org_user_level ON logs (organization_id, user_id, level, timestamp DESC) WHERE user_id IS NOT NULL;

-- Error-specific index for debugging
CREATE INDEX idx_logs_errors ON logs (organization_id, timestamp DESC) WHERE level IN ('ERROR', 'FATAL');

-- Create trigger to automatically update message_vector for full-text search
CREATE OR REPLACE FUNCTION update_logs_search_vector()
RETURNS TRIGGER AS $$
BEGIN
    NEW.message_vector := to_tsvector('english', COALESCE(NEW.message, '') || ' ' || 
                                     COALESCE(NEW.component, '') || ' ' || 
                                     COALESCE(NEW.plugin_name, '') || ' ' || 
                                     COALESCE(NEW.action, '') || ' ' ||
                                     COALESCE(NEW.error_message, ''));
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_logs_search_vector
    BEFORE INSERT OR UPDATE ON logs
    FOR EACH ROW
    EXECUTE FUNCTION update_logs_search_vector();

-- Log statistics table for monitoring and analytics
CREATE TABLE log_statistics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL,
    date DATE NOT NULL,
    component VARCHAR(100) NOT NULL,
    level log_level NOT NULL,
    count BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Ensure one record per org/date/component/level combination
    CONSTRAINT unique_log_stats UNIQUE (organization_id, date, component, level)
);

-- Convert statistics to hypertable with daily partitioning
-- Note: Commented out for PostgreSQL-only setup. Uncomment when TimescaleDB is available.
-- SELECT create_hypertable('log_statistics', 'date', chunk_time_interval => INTERVAL '7 days');

-- Statistics indexes
CREATE INDEX idx_log_stats_org_date ON log_statistics (organization_id, date DESC);
CREATE INDEX idx_log_stats_component ON log_statistics (component, date DESC);

-- Function to update statistics (called by application or cron job)
CREATE OR REPLACE FUNCTION update_log_statistics(target_date DATE DEFAULT CURRENT_DATE)
RETURNS VOID AS $$
BEGIN
    INSERT INTO log_statistics (organization_id, date, component, level, count, updated_at)
    SELECT 
        organization_id,
        target_date,
        component,
        level,
        COUNT(*) as count,
        NOW() as updated_at
    FROM logs 
    WHERE DATE(timestamp) = target_date
    GROUP BY organization_id, component, level
    ON CONFLICT (organization_id, date, component, level) 
    DO UPDATE SET 
        count = EXCLUDED.count,
        updated_at = EXCLUDED.updated_at;
END;
$$ LANGUAGE plpgsql;

-- Data retention policy (follows guidelines for storage growth planning)
-- Automatically delete logs older than 90 days (configurable)
-- TimescaleDB compression for older data (keep 30 days uncompressed)

-- Enable compression for chunks older than 30 days
-- Note: Commented out for PostgreSQL-only setup. Uncomment when TimescaleDB is available.
-- SELECT add_compression_policy('logs', INTERVAL '30 days');

-- Data retention policy - delete data older than 90 days
-- Note: Commented out for PostgreSQL-only setup. Uncomment when TimescaleDB is available.
-- SELECT add_retention_policy('logs', INTERVAL '90 days');

-- Statistics retention - keep for 1 year
-- Note: Commented out for PostgreSQL-only setup. Uncomment when TimescaleDB is available.
-- SELECT add_retention_policy('log_statistics', INTERVAL '1 year');

-- Create view for common log queries (performance optimization)
CREATE VIEW logs_with_user_info AS
SELECT 
    l.*,
    u.email as user_email,
    u.first_name || ' ' || u.last_name as user_name,
    o.name as organization_name
FROM logs l
LEFT JOIN users u ON l.user_id = u.id
LEFT JOIN organizations o ON l.organization_id = o.id;

-- Create materialized view for dashboard metrics (refreshed periodically)
CREATE MATERIALIZED VIEW log_metrics AS
SELECT 
    organization_id,
    component,
    level,
    DATE(timestamp) as log_date,
    COUNT(*) as total_count,
    COUNT(DISTINCT user_id) as unique_users,
    COUNT(DISTINCT session_id) as unique_sessions,
    AVG(char_length(message)) as avg_message_length
FROM logs 
WHERE timestamp >= CURRENT_DATE - INTERVAL '30 days'
GROUP BY organization_id, component, level, DATE(timestamp);

-- Index for materialized view
CREATE INDEX idx_log_metrics_org_date ON log_metrics (organization_id, log_date DESC);

-- Refresh function for materialized view (to be called by scheduler)
CREATE OR REPLACE FUNCTION refresh_log_metrics()
RETURNS VOID AS $$
BEGIN
    REFRESH MATERIALIZED VIEW log_metrics;
END;
$$ LANGUAGE plpgsql;

-- Log levels for reference (matching Go constants)
-- DEBUG: Detailed information for debugging
-- INFO:  General information about program execution
-- WARN:  Warning messages for potentially harmful situations
-- ERROR: Error events that might still allow application to continue
-- FATAL: Severe error events that will lead to application termination

-- Performance optimization: Log partitioning examples for very high volume
-- Additional partitioning by organization for large multi-tenant deployments
-- (Only needed if organization count is high and log volume is extreme)

/*
-- Example: Additional partitioning by organization (for extreme scale)
CREATE TABLE logs_template (LIKE logs);
CREATE TABLE logs_org_001 PARTITION OF logs_template 
    FOR VALUES FROM ('00000000-0000-0000-0000-000000000001') 
                  TO ('99999999-9999-9999-9999-999999999999');
*/

-- Security: Row Level Security for multi-tenant access
-- This ensures users can only access logs from their organization
ALTER TABLE logs ENABLE ROW LEVEL SECURITY;

-- Policy: Users can only see logs from their organization
CREATE POLICY logs_org_isolation ON logs
    FOR ALL
    TO PUBLIC
    USING (organization_id = current_setting('app.current_organization_id')::UUID);

-- Policy for system administrators (can see all organizations)
CREATE POLICY logs_system_admin ON logs
    FOR ALL
    TO PUBLIC
    USING (current_setting('app.is_system_admin', true)::BOOLEAN = true);

-- Apply same security to statistics
ALTER TABLE log_statistics ENABLE ROW LEVEL SECURITY;

CREATE POLICY log_stats_org_isolation ON log_statistics
    FOR ALL
    TO PUBLIC
    USING (organization_id = current_setting('app.current_organization_id')::UUID);

CREATE POLICY log_stats_system_admin ON log_statistics
    FOR ALL
    TO PUBLIC
    USING (current_setting('app.is_system_admin', true)::BOOLEAN = true);

-- Comments for documentation
COMMENT ON TABLE logs IS 'Time-series log storage with TimescaleDB optimization. Stores all application logs with multi-tenant isolation and full-text search capabilities.';
COMMENT ON COLUMN logs.organization_id IS 'Multi-tenant isolation field - all logs must belong to an organization';
COMMENT ON COLUMN logs.details IS 'JSONB field for flexible metadata storage, indexed for complex queries';
COMMENT ON COLUMN logs.message_vector IS 'Automatically maintained tsvector for full-text search on log messages';
COMMENT ON TABLE log_statistics IS 'Aggregated log statistics for monitoring and dashboard metrics';

-- Example queries for testing and documentation

/*
-- Query examples:

-- 1. Recent errors for an organization
SELECT * FROM logs 
WHERE organization_id = $1 
  AND level IN ('ERROR', 'FATAL') 
  AND timestamp >= NOW() - INTERVAL '24 hours'
ORDER BY timestamp DESC;

-- 2. User activity logs
SELECT * FROM logs_with_user_info 
WHERE organization_id = $1 
  AND user_id = $2 
  AND timestamp >= NOW() - INTERVAL '7 days'
ORDER BY timestamp DESC;

-- 3. Full-text search across messages
SELECT * FROM logs 
WHERE organization_id = $1 
  AND message_vector @@ plainto_tsquery('english', 'build failed')
ORDER BY timestamp DESC;

-- 4. Plugin-specific logs
SELECT * FROM logs 
WHERE organization_id = $1 
  AND plugin_name = 'auth-plugin'
  AND timestamp >= NOW() - INTERVAL '1 hour'
ORDER BY timestamp DESC;

-- 5. Component statistics for dashboard
SELECT component, level, COUNT(*) as count
FROM logs 
WHERE organization_id = $1 
  AND timestamp >= CURRENT_DATE
GROUP BY component, level
ORDER BY component, level;

-- 6. Performance monitoring - slow operations
SELECT * FROM logs 
WHERE organization_id = $1 
  AND details->>'duration_ms' IS NOT NULL
  AND (details->>'duration_ms')::INTEGER > 1000
  AND timestamp >= NOW() - INTERVAL '1 hour'
ORDER BY (details->>'duration_ms')::INTEGER DESC;
*/