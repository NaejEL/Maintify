# Logging API Integration Guide

> **Status**: API Implementation Complete, Routes Not Registered
>
> This guide describes the logging API that will be available once integration is complete. The service code exists in `/core/pkg/logging/` but routes are not yet registered in `main.go`. See [ROADMAP.md](../ROADMAP.md) Priority 1 for integration timeline.

This guide explains how third-party plugin developers will integrate with Maintify's logging system via HTTP APIs.

## Overview

Maintify provides a comprehensive logging API that allows plugins to:

- Submit structured logs with metadata
- Search and retrieve historical logs
- Access aggregated statistics and metrics
- Manage log retention policies (admin only)

All logging operations are secured with RBAC authentication and scoped to your organization.

## Authentication

All logging API endpoints require authentication via JWT tokens obtained from the RBAC system.

### Required Headers

```http
Authorization: Bearer <jwt-token>
X-Organization-ID: <organization-uuid>
Content-Type: application/json
```

### Optional Context Headers

```http
X-User-ID: <user-uuid>          # Auto-populated if not provided
X-Session-ID: <session-id>      # Auto-populated if not provided  
X-Request-ID: <request-id>      # Auto-populated if not provided
```

## API Endpoints

### Base URL

```text
https://your-maintify-instance.com/api/logs
```

### 1. Log Ingestion

Submit logs in batches for efficient processing.

**Endpoint:** `POST /api/logs`
**Permission:** `logs:write`

#### Request Body

```json
{
  "entries": [
    {
      "level": "INFO",
      "component": "auth-plugin",
      "message": "User login successful",
      "action": "user_login",
      "user_id": "123e4567-e89b-12d3-a456-426614174000",
      "session_id": "sess_abc123",
      "request_id": "req_xyz789",
      "metadata": {
        "ip_address": "192.168.1.100",
        "user_agent": "Mozilla/5.0...",
        "login_method": "password"
      }
    }
  ],
  "batch_id": "batch_abc123",
  "source": "auth-plugin-v1.2.0"
}
```

#### Log Entry Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `level` | string | Yes | `DEBUG`, `INFO`, `WARN`, `ERROR`, `FATAL` |
| `component` | string | Yes | Component/module name (max 100 chars) |
| `message` | string | Yes | Human-readable log message |
| `action` | string | No | Action being performed |
| `user_id` | UUID | No | User associated with the action |
| `session_id` | string | No | Session identifier |
| `request_id` | string | No | Request correlation ID |
| `plugin_name` | string | No | Plugin identifier |
| `metadata` | object | No | Additional structured data |

#### Success Response

```json
{
  "success": true,
  "processed_count": 1,
  "failed_count": 0,
  "batch_id": "batch_abc123",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

#### Error Response

```json
{
  "success": false,
  "processed_count": 0,
  "failed_count": 1,
  "errors": {
    "0": {
      "code": "validation_failed",
      "message": "Missing required field: level",
      "field": "level"
    }
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### 2. Log Search

Search and retrieve logs with flexible filtering.

**Endpoint:** `GET /api/logs/search`
**Permission:** `logs:read`

#### Search Query Parameters

| Parameter | Type | Description | Example |
|-----------|------|-------------|---------|
| `start_time` | ISO8601 | Filter logs after this time | `2024-01-15T00:00:00Z` |
| `end_time` | ISO8601 | Filter logs before this time | `2024-01-15T23:59:59Z` |
| `since` | duration | Relative time filter | `1h`, `24h`, `7d` |
| `levels` | string | Comma-separated log levels | `INFO,WARN,ERROR` |
| `components` | string | Comma-separated components | `auth,payment` |
| `user_id` | UUID | Filter by user ID | `123e4567-e89b-...` |
| `plugin_name` | string | Filter by plugin name | `auth-plugin` |
| `action` | string | Filter by action | `user_login` |
| `q` | string | Full-text search | `login failed` |
| `limit` | integer | Results per page (max 1000) | `50` |
| `offset` | integer | Results offset | `100` |

#### Search Response

```json
{
  "entries": [
    {
      "id": "log_456",
      "level": "INFO",
      "component": "auth-plugin",
      "message": "User login successful",
      "action": "user_login",
      "user_id": "123e4567-e89b-12d3-a456-426614174000",
      "session_id": "sess_abc123",
      "request_id": "req_xyz789",
      "plugin_name": "auth-plugin",
      "metadata": {
        "ip_address": "192.168.1.100",
        "login_method": "password"
      },
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T10:30:00Z",
      "user_email": "john@example.com",
      "user_name": "John Doe"
    }
  ],
  "pagination": {
    "limit": 50,
    "offset": 0,
    "total_count": 1,
    "returned_count": 1,
    "has_more": false,
    "next_offset": null,
    "previous_offset": null
  },
  "search_meta": {
    "execution_time": "15ms",
    "filters_applied": {
      "levels": ["INFO"],
      "time_range": "2024-01-15T00:00:00Z to 2024-01-15T23:59:59Z"
    }
  },
  "timestamp": "2024-01-15T11:00:00Z"
}
```

### 3. Log Statistics

Get aggregated metrics and statistics.

**Endpoint:** `GET /api/logs/statistics`
**Permission:** `logs:read`

#### Statistics Query Parameters

| Parameter | Type | Description | Default |
|-----------|------|-------------|---------|
| `start_time` | ISO8601 | Statistics start time | `2024-01-15T00:00:00Z` |
| `end_time` | ISO8601 | Statistics end time | `2024-01-15T23:59:59Z` |
| `since` | duration | Relative time range | `24h`, `7d`, `30d` |

#### Statistics Response

```json
{
  "total_logs": 15420,
  "level_counts": {
    "DEBUG": 8200,
    "INFO": 6000,
    "WARN": 1000,
    "ERROR": 200,
    "FATAL": 20
  },
  "component_counts": {
    "auth-plugin": 5000,
    "payment-plugin": 3000,
    "core": 7420
  },
  "plugin_counts": {
    "auth-plugin": 5000,
    "payment-plugin": 3000
  },
  "time_series": [
    {
      "timestamp": "2024-01-15T00:00:00Z",
      "count": 500,
      "breakdown": {
        "INFO": 400,
        "WARN": 80,
        "ERROR": 20
      }
    }
  ],
  "top_errors": [
    {
      "message": "Database connection timeout",
      "count": 15,
      "first_seen": "2024-01-15T08:00:00Z",
      "last_seen": "2024-01-15T10:30:00Z"
    }
  ],
  "average_logs_per_hour": 642.5,
  "peak_hour": "2024-01-15T09:00:00Z",
  "time_range": {
    "start": "2024-01-15T00:00:00Z",
    "end": "2024-01-15T23:59:59Z"
  },
  "timestamp": "2024-01-15T11:00:00Z"
}
```

### 4. Log Cleanup (Admin Only)

Remove old logs based on retention policies.

**Endpoint:** `DELETE /api/logs/admin/cleanup`
**Permission:** `logs:admin`

#### Query Parameters

| Parameter | Type | Description | Default |
|-----------|------|-------------|---------|
| `older_than` | duration | Delete logs older than this | `90d` |

#### Response

```json
{
  "success": true,
  "deleted_count": 50000,
  "retention": "90d",
  "timestamp": "2024-01-15T11:00:00Z"
}
```

## Integration Examples

### JavaScript/Node.js

```javascript
class MaintifyLogger {
  constructor(baseUrl, token, organizationId) {
    this.baseUrl = baseUrl;
    this.token = token;
    this.organizationId = organizationId;
  }

  async log(level, component, message, options = {}) {
    const entry = {
      level: level.toUpperCase(),
      component,
      message,
      ...options
    };

    const response = await fetch(`${this.baseUrl}/api/logs`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${this.token}`,
        'X-Organization-ID': this.organizationId,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        entries: [entry],
        source: 'my-plugin-v1.0.0'
      })
    });

    if (!response.ok) {
      throw new Error(`Logging failed: ${response.statusText}`);
    }

    return await response.json();
  }

  async search(filters = {}) {
    const params = new URLSearchParams(filters);
    const response = await fetch(`${this.baseUrl}/api/logs/search?${params}`, {
      headers: {
        'Authorization': `Bearer ${this.token}`,
        'X-Organization-ID': this.organizationId
      }
    });

    if (!response.ok) {
      throw new Error(`Search failed: ${response.statusText}`);
    }

    return await response.json();
  }
}

// Usage
const logger = new MaintifyLogger(
  'https://maintify.example.com',
  'your-jwt-token',
  'your-org-id'
);

// Log an event
await logger.log('INFO', 'payment-plugin', 'Payment processed successfully', {
  action: 'process_payment',
  metadata: {
    amount: 99.99,
    currency: 'USD',
    payment_method: 'credit_card'
  }
});

// Search logs
const results = await logger.search({
  since: '1h',
  levels: 'ERROR,FATAL',
  component: 'payment-plugin'
});
```

### Python

```python
import requests
import json
from datetime import datetime, timedelta

class MaintifyLogger:
    def __init__(self, base_url, token, organization_id):
        self.base_url = base_url
        self.token = token
        self.organization_id = organization_id
        self.session = requests.Session()
        self.session.headers.update({
            'Authorization': f'Bearer {token}',
            'X-Organization-ID': organization_id,
            'Content-Type': 'application/json'
        })

    def log(self, level, component, message, **kwargs):
        entry = {
            'level': level.upper(),
            'component': component,
            'message': message,
            **kwargs
        }

        payload = {
            'entries': [entry],
            'source': 'my-python-plugin-v1.0.0'
        }

        response = self.session.post(
            f'{self.base_url}/api/logs',
            json=payload
        )
        response.raise_for_status()
        return response.json()

    def search(self, **filters):
        response = self.session.get(
            f'{self.base_url}/api/logs/search',
            params=filters
        )
        response.raise_for_status()
        return response.json()

    def get_statistics(self, since='24h'):
        response = self.session.get(
            f'{self.base_url}/api/logs/statistics',
            params={'since': since}
        )
        response.raise_for_status()
        return response.json()

# Usage
logger = MaintifyLogger(
    'https://maintify.example.com',
    'your-jwt-token',
    'your-org-id'
)

# Log an error
logger.log('ERROR', 'auth-plugin', 'Failed to authenticate user', 
           action='authenticate',
           metadata={'reason': 'invalid_credentials', 'attempts': 3})

# Search recent errors
errors = logger.search(since='1h', levels='ERROR,FATAL')
print(f"Found {errors['pagination']['total_count']} errors")
```

### Go

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

type MaintifyLogger struct {
    BaseURL        string
    Token          string
    OrganizationID string
    Client         *http.Client
}

type LogEntry struct {
    Level     string                 `json:"level"`
    Component string                 `json:"component"`
    Message   string                 `json:"message"`
    Action    string                 `json:"action,omitempty"`
    Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type LogRequest struct {
    Entries []LogEntry `json:"entries"`
    Source  string     `json:"source"`
}

func NewMaintifyLogger(baseURL, token, orgID string) *MaintifyLogger {
    return &MaintifyLogger{
        BaseURL:        baseURL,
        Token:          token,
        OrganizationID: orgID,
        Client:         &http.Client{Timeout: 30 * time.Second},
    }
}

func (m *MaintifyLogger) Log(level, component, message string, metadata map[string]interface{}) error {
    entry := LogEntry{
        Level:     level,
        Component: component,
        Message:   message,
        Metadata:  metadata,
    }

    request := LogRequest{
        Entries: []LogEntry{entry},
        Source:  "my-go-plugin-v1.0.0",
    }

    jsonData, _ := json.Marshal(request)
    
    req, _ := http.NewRequest("POST", m.BaseURL+"/api/logs", bytes.NewBuffer(jsonData))
    req.Header.Set("Authorization", "Bearer "+m.Token)
    req.Header.Set("X-Organization-ID", m.OrganizationID)
    req.Header.Set("Content-Type", "application/json")

    resp, err := m.Client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusCreated {
        return fmt.Errorf("logging failed with status: %d", resp.StatusCode)
    }

    return nil
}

// Usage
func main() {
    logger := NewMaintifyLogger(
        "https://maintify.example.com",
        "your-jwt-token",
        "your-org-id",
    )

    // Log an event
    err := logger.Log("INFO", "user-plugin", "User created successfully", 
        map[string]interface{}{
            "user_id": "123",
            "role":    "admin",
        })
    
    if err != nil {
        panic(err)
    }
}
```

## Best Practices

### 1. Efficient Logging

- **Batch logs**: Submit multiple entries in a single request (up to 100)
- **Use structured metadata**: Store additional context in the metadata field
- **Set appropriate log levels**: Use DEBUG sparingly, INFO for normal operations
- **Include correlation IDs**: Use request_id and session_id for tracing

### 2. Error Handling

- **Handle rate limits**: Implement exponential backoff for retries
- **Validate before sending**: Check required fields locally first
- **Monitor failed requests**: Track ingestion failures and retry logic

### 3. Security

- **Secure token storage**: Store JWT tokens securely, rotate regularly
- **Filter sensitive data**: Never log passwords, API keys, or PII
- **Use organization scoping**: Ensure logs are properly scoped to your organization

### 4. Performance

- **Async logging**: Don't block business logic on log submission
- **Cache statistics**: Cache frequently accessed metrics
- **Use search filters**: Apply specific filters to reduce result sets

## Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `invalid_json` | 400 | Request body is not valid JSON |
| `validation_failed` | 400 | Required fields missing or invalid |
| `unauthorized` | 401 | Invalid or missing authentication |
| `forbidden` | 403 | Insufficient permissions |
| `too_many_entries` | 400 | More than 100 entries in batch |
| `ingestion_failed` | 500 | Database or internal error |
| `search_failed` | 500 | Search operation failed |

## Rate Limits

- **Log ingestion**: 1000 requests/hour per organization
- **Search requests**: 500 requests/hour per user
- **Statistics**: 100 requests/hour per user

Contact your Maintify administrator to adjust rate limits if needed.

## Support

For integration support:

- Review the [API documentation](../docs/LOGGING_SYSTEM_DESIGN.md)
- Check the [development guidelines](../GUIDELINES.md)
- Contact your system administrator for access issues
