# Plugin Sidecar Pattern Guide

> **Status**: Design Complete, Implementation Pending
>
> This document describes the sidecar pattern for plugins requiring specialized databases. No plugins have been implemented yet. See [ROADMAP.md](../ROADMAP.md) Phase 2 for plugin system development timeline.

This guide explains how to create plugins that require specialized databases or services using the "sidecar" pattern in Maintify.

## Overview

The sidecar pattern allows plugins to bring their own dependencies (databases, message queues, etc.) without affecting the core system. Each plugin that needs specialized data storage manages its own infrastructure as companion containers.

## When to Use Sidecar Services

Use the sidecar pattern when your plugin needs:
- **Specialized databases** (Elasticsearch, InfluxDB, MongoDB, etc.)
- **Message queues** (RabbitMQ, Apache Kafka)
- **Cache layers** (Memcached, specialized Redis instances)
- **Search engines** (Solr, Typesense)
- **Time-series databases** (TimescaleDB, ClickHouse)

## Implementation Steps

### 1. Create Plugin Structure

```
plugins/your-plugin/
├── backend/
│   ├── Dockerfile
│   ├── main.go
│   └── go.mod
├── frontend/
│   └── ... (Flutter files)
├── plugin.yaml
└── docker-compose.plugin.yml    # ← New sidecar definition
```

### 2. Define Sidecar Services

Create `docker-compose.plugin.yml` in your plugin's root directory:

```yaml
# Example: Logging plugin with Elasticsearch
version: '3.8'
services:
  elasticsearch:
    image: elasticsearch:8.11.0
    container_name: maintify-logging-elasticsearch
    environment:
      - discovery.type=single-node
      - xpack.security.enabled=false
      - "ES_JAVA_OPTS=-Xms512m -Xmx512m"
    ports:
      - "9200:9200"
    volumes:
      - elasticsearch_data:/usr/share/elasticsearch/data
    networks:
      - maintify_network

  kibana:
    image: kibana:8.11.0
    container_name: maintify-logging-kibana
    environment:
      - ELASTICSEARCH_HOSTS=http://elasticsearch:9200
    ports:
      - "5601:5601"
    depends_on:
      - elasticsearch
    networks:
      - maintify_network

volumes:
  elasticsearch_data:

networks:
  maintify_network:
    external: true
```

### 3. Configure Plugin Backend

Update your plugin's backend to connect to the sidecar service:

```go
// backend/main.go
package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "github.com/elastic/go-elasticsearch/v8"
)

func main() {
    // Connect to Elasticsearch sidecar
    esConfig := elasticsearch.Config{
        Addresses: []string{
            os.Getenv("ELASTICSEARCH_URL"), // Default: http://elasticsearch:9200
        },
    }
    
    es, err := elasticsearch.NewClient(esConfig)
    if err != nil {
        log.Fatalf("Failed to connect to Elasticsearch: %v", err)
    }
    
    // Test connection
    res, err := es.Info()
    if err != nil {
        log.Fatalf("Failed to get Elasticsearch info: %v", err)
    }
    defer res.Body.Close()
    
    log.Println("Successfully connected to Elasticsearch sidecar")
    
    // Start your plugin HTTP server
    http.HandleFunc("/logs", handleLogs)
    log.Fatal(http.ListenAndServe(":8093", nil))
}

func handleLogs(w http.ResponseWriter, r *http.Request) {
    // Your logging logic here
}
```

### 4. Update Plugin Metadata

Add sidecar information to `plugin.yaml`:

```yaml
name: logging
description: Advanced logging plugin with Elasticsearch
version: 1.0.0
backend_url: http://logging:8093
frontend_url: /logging
route: /api/logging
schema: logging

# Resource limits for the plugin itself
resources:
  memory_mb: 256
  cpu_milli_cores: 500

# Sidecar services configuration
sidecar:
  enabled: true
  compose_file: docker-compose.plugin.yml
  services:
    - name: elasticsearch
      health_check: "curl -f http://elasticsearch:9200/_cluster/health"
      startup_timeout: 60s
    - name: kibana
      health_check: "curl -f http://kibana:5601/api/status"
      startup_timeout: 120s
```

## Development Workflow

### Running Locally

1. **Start Core Services**:
   ```bash
   docker-compose up --build core builder db redis
   ```

2. **Start Plugin Sidecar Services**:
   ```bash
   cd plugins/your-plugin
   docker-compose -f docker-compose.plugin.yml up -d
   ```

3. **Build and Test Plugin**:
   ```bash
   # The core will automatically discover and build your plugin
   # The plugin will connect to the sidecar services
   ```

### Environment Variables

Set these in your plugin's backend environment:

```yaml
# In your plugin's container environment
environment:
  - ELASTICSEARCH_URL=http://elasticsearch:9200
  - KIBANA_URL=http://kibana:5601
  - REDIS_URL=redis://redis:6379  # Shared Redis for hooks
```

## Best Practices

### 1. Network Configuration
- Always use the external `maintify_network` so services can communicate
- Use service names for internal communication (e.g., `elasticsearch:9200`)

### 2. Data Persistence
- Always define volumes for data that should persist
- Use descriptive volume names: `{plugin-name}_{service}_data`

### 3. Resource Management
- Set appropriate memory limits for sidecar services
- Monitor resource usage, especially for data stores

### 4. Health Checks
- Implement health checks for all sidecar services
- Define reasonable startup timeouts

### 5. Security
- Use non-root users in sidecar containers when possible
- Disable unnecessary features (like X-Pack security in dev Elasticsearch)
- Consider network policies for production

## Examples

### Time-Series Plugin with InfluxDB

```yaml
# docker-compose.plugin.yml
version: '3.8'
services:
  influxdb:
    image: influxdb:2.7
    container_name: maintify-metrics-influxdb
    environment:
      - DOCKER_INFLUXDB_INIT_MODE=setup
      - DOCKER_INFLUXDB_INIT_USERNAME=admin
      - DOCKER_INFLUXDB_INIT_PASSWORD=password123
      - DOCKER_INFLUXDB_INIT_ORG=maintify
      - DOCKER_INFLUXDB_INIT_BUCKET=metrics
    ports:
      - "8086:8086"
    volumes:
      - influxdb_data:/var/lib/influxdb2
    networks:
      - maintify_network

volumes:
  influxdb_data:

networks:
  maintify_network:
    external: true
```

### Analytics Plugin with ClickHouse

```yaml
# docker-compose.plugin.yml
version: '3.8'
services:
  clickhouse:
    image: clickhouse/clickhouse-server:23.8
    container_name: maintify-analytics-clickhouse
    environment:
      - CLICKHOUSE_DB=analytics
      - CLICKHOUSE_USER=maintify
      - CLICKHOUSE_DEFAULT_ACCESS_MANAGEMENT=1
      - CLICKHOUSE_PASSWORD=secure_password
    ports:
      - "8123:8123"
      - "9000:9000"
    volumes:
      - clickhouse_data:/var/lib/clickhouse
    networks:
      - maintify_network

volumes:
  clickhouse_data:

networks:
  maintify_network:
    external: true
```

## Troubleshooting

### Common Issues

1. **Network Connectivity**:
   - Ensure all services use `maintify_network`
   - Use service names, not localhost

2. **Service Discovery**:
   - Wait for health checks to pass before connecting
   - Implement retry logic in your plugin

3. **Data Persistence**:
   - Always define volumes for data directories
   - Check volume mount permissions

4. **Resource Limits**:
   - Monitor memory usage of sidecar services
   - Adjust limits based on your data volume

### Debugging Commands

```bash
# Check network connectivity
docker network ls
docker network inspect maintify_default

# Check service logs
docker-compose -f docker-compose.plugin.yml logs elasticsearch

# Test service connectivity
docker exec maintify-plugin-container-logging curl -f http://elasticsearch:9200
```

This pattern ensures your plugins are self-contained while maintaining the security and isolation principles of the Maintify architecture.