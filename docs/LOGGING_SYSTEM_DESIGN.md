# Maintify Logging System Design

> **Status**: Implementation Complete (Code 100%), Integration Pending (0%)
>
> This document describes the logging system architecture. The code is fully implemented in `/core/pkg/logging/` but not yet integrated into the main application. See [ROADMAP.md](../ROADMAP.md) for integration timeline.

## Overview

The Maintify logging system is designed with an **API-first approach** following the project guidelines. This ensures third-party plugin developers can submit logs through HTTP APIs without requiring Go package dependencies.

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Core Service  │───▶│  TimescaleDB     │◄───│  Log Search API │
│   (generates    │    │  (time-series    │    │  (RBAC-secured  │
│    logs)        │    │   storage)       │    │   access)       │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                        ▲                       │
         ▼                        │                       ▼
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Plugin Logs   │───▶│  Log Ingestion   │    │   Admin UI      │
│   (via HTTP API)│    │  API             │    │   (log viewer)  │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

## Database Schema Features

### TimescaleDB Optimization
- **Time-series partitioning**: 1-day chunks for optimal performance
- **Automatic compression**: Data older than 30 days compressed
- **Data retention**: Automatic cleanup after 90 days (configurable)
- **Performance**: Designed for 10K-100K entries/day per guidelines

### Multi-Tenant Security
- **Organization isolation**: All logs scoped to organization_id
- **Row Level Security**: PostgreSQL RLS prevents cross-tenant access
- **RBAC integration**: Permission-controlled log access
- **JWT-based auth**: Consistent with existing authentication

### Search Capabilities
- **Full-text search**: Automatic tsvector generation for message content
- **JSONB metadata**: Flexible details field with GIN indexing
- **Time-range queries**: Optimized timestamp-based filtering
- **Multi-field search**: Component, level, user, plugin filtering

## API Design (HTTP-First for Third-Party Developers)

### Log Submission API

```http
POST /api/logs
Authorization: Bearer {jwt-token}
Content-Type: application/json

{
  "entries": [
    {
      "level": "INFO",
      "message": "User action completed successfully",
      "component": "auth-plugin",
      "user_id": "user-uuid",
      "session_id": "session-123",
      "request_id": "req-456",
      "plugin_name": "auth-plugin",
      "action": "user_login",
      "details": {
        "ip_address": "192.168.1.100",
        "user_agent": "Mozilla/5.0...",
        "duration_ms": 150
      }
    }
  ]
}
```

### Log Search API

```http
GET /api/logs/search?level=ERROR&component=auth-plugin&since=2h
Authorization: Bearer {jwt-token}

Response:
{
  "logs": [...],
  "total": 42,
  "page": 1,
  "per_page": 50
}
```

### Plugin Integration Example

```javascript
// Third-party plugin logging (any language with HTTP client)
const logClient = {
  baseURL: 'http://core:8080/api/logs',
  token: 'jwt-token-from-auth',
  
  async log(level, message, details = {}) {
    const entry = {
      level: level.toUpperCase(),
      message,
      component: 'my-custom-plugin',
      plugin_name: 'my-custom-plugin',
      details
    };
    
    await fetch(this.baseURL, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${this.token}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ entries: [entry] })
    });
  }
};

// Usage in plugin
await logClient.log('INFO', 'Processing maintenance request', {
  equipment_id: 'eq-123',
  user_id: 'user-456',
  action: 'schedule_maintenance'
});
```

## Benefits of API-First Approach

### For Third-Party Developers
- **Language Agnostic**: Any language with HTTP client support
- **No Go Dependencies**: No need to import Maintify Go packages
- **Simple Integration**: Standard REST API calls
- **Consistent Authentication**: Same JWT tokens as other APIs

### For Maintify Core
- **Centralized Logging**: All logs flow through single ingestion point
- **Security Control**: RBAC permissions apply to all log operations
- **Performance Optimization**: Batch processing and validation
- **Audit Trail**: Complete logging of all plugin activities

## Integration with Existing Logger

The existing Go logger package will be enhanced to support dual output:

1. **Existing Outputs**: Console and file logging (unchanged)
2. **New Database Output**: HTTP calls to log ingestion API
3. **Backward Compatibility**: No breaking changes to existing code

```go
// Enhanced logger will automatically send to database
logger.Info("User logged in", map[string]interface{}{
    "user_id": userID,
    "session_id": sessionID,
})
// This will now:
// 1. Write to console/file (existing behavior)
// 2. Send HTTP request to /api/logs (new behavior)
```

## RBAC Integration

### Permission Model
- `logs:read` - View logs for accessible resources
- `logs:read:all` - View all organization logs (admin)
- `logs:write` - Submit logs (plugin service accounts)
- `logs:admin` - Manage log retention and settings

### Access Control
- **Organization Scoping**: Users only see logs from their organization
- **Resource Filtering**: Logs filtered by user's resource permissions
- **Plugin Isolation**: Plugins can only submit logs, not read others' logs

## Performance Considerations

### Database Performance
- **Indexes**: Optimized for common query patterns
- **Partitioning**: TimescaleDB time-based partitioning
- **Statistics**: Pre-aggregated metrics for dashboards
- **Connection Pooling**: Efficient database connection management

### API Performance
- **Batch Processing**: Multiple log entries per API call
- **Async Processing**: Non-blocking log submission
- **Rate Limiting**: Prevent log spam attacks
- **Caching**: Frequently accessed log data cached

## Monitoring and Alerting

### Log Statistics
- **Volume Metrics**: Logs per component/hour/day
- **Error Rates**: Track error/fatal log percentages
- **Performance**: API response times and database performance
- **Storage Growth**: Monitor disk usage and retention

### Health Checks
- **Database Connectivity**: TimescaleDB connection status
- **API Responsiveness**: Log ingestion API health
- **Disk Space**: Storage capacity monitoring
- **Retention Policies**: Automatic cleanup verification

## Future Enhancements

### Phase 2 Features
- **Real-time Streaming**: WebSocket API for live log viewing
- **Advanced Analytics**: Log pattern analysis and anomaly detection
- **Export Capabilities**: Bulk log export for compliance
- **Integration APIs**: Webhook notifications for critical events

### Scalability Planning
- **Horizontal Scaling**: Multiple log ingestion instances
- **Sharding**: Database sharding for extreme scale
- **Compression**: Advanced compression for long-term storage
- **Archival**: Cold storage for compliance retention

This design ensures that the logging system follows Maintify's guidelines while providing a developer-friendly, scalable, and secure logging infrastructure for the entire platform.