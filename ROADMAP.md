# Maintify Development Roadmap

**Status**: Early Development - Core Infrastructure Phase

This roadmap outlines the development plan for Maintify, a plugin-based CMMS platform. The project is currently in early development, focusing on building robust core infrastructure before expanding to the full plugin ecosystem.

> **Development Methodology**: Test-Driven Development (TDD)
>
> All features are developed using TDD methodology: Write test → Fail → Implement → Pass → Refactor. Every code change must include both unit tests and integration tests. Security scanning and static analysis are mandatory before any merge.

---

## 🎯 Project Vision

Build an enterprise-grade, plugin-based Computerized Maintenance Management System (CMMS) that scales from small businesses to large enterprises, with:
- **Security-first design** with sandboxed plugin execution and mandatory security scanning
- **Test-Driven Development** with comprehensive unit and integration test coverage
- **Multi-tenant RBAC** with organization isolation and audit trails
- **Containerized plugin architecture** for extensibility and isolation
- **Zero technical debt** through continuous static analysis and code quality gates

---

## 📊 Current Status: Phase 1 - Core Infrastructure

**Overall Progress**: ✅ Phase 1 Complete

### What's Working Today ✅

**RBAC System** (100% Complete)
- ✅ Multi-tenant organization support with complete data isolation
- ✅ User authentication with JWT tokens and bcrypt password hashing
- ✅ Role-based permission system with resource-level access control
- ✅ Time-based access control (scheduled role assignments)
- ✅ Emergency access workflows with break-glass procedures
- ✅ Comprehensive audit logging for all RBAC operations
- ✅ PostgreSQL database with 16 tables + 3 views + 6 functions
- ✅ Full REST API with authentication middleware
- ✅ Background processors for time-based and emergency access

**Core Service** (80% Complete)
- ✅ HTTP server with health check endpoints
- ✅ Database connection pooling and configuration
- ✅ Database migration system with auto-apply
- ✅ Graceful shutdown with signal handling
- ✅ CORS middleware for development
- ✅ Structured logging to console and files
- ✅ Logging service integrated and verified with database storage
- ✅ Logging package test suite passing in current core test run (`go test ./...`)
- ✅ Plugin manager initialized at startup
- ✅ Plugin lifecycle HTTP API fully implemented and tested (start/stop/restart/status/diagnostics)
- ✅ Plugin health monitoring integrated into diagnostics endpoint

**Builder Service** (90% Complete)
- ✅ Docker image building via API
- ✅ API key authentication
- ✅ Build context handling and image tagging
- ✅ Structured logging
- ✅ Image cleanup enabled and working (Docker SDK v27 types fixed)

**Frontend** (Demo Stage)
- ✅ React + Vite setup with login page
- ✅ Dashboard UI for testing RBAC
- ✅ JWT token management
- ✅ Docker deployment with Nginx
- ⚠️ Basic demo only - will be enhanced after backend completion

**Infrastructure**
- ✅ Docker Compose orchestration for all services
- ✅ PostgreSQL database with proper schema
- ✅ Redis container running (used by hooks foundation, no plugin traffic yet)
- ✅ Multi-service networking configured

### What's In Progress 🔄

**Plugin System** (Architecture: 100%, Implementation: 97%)
- ✅ Docker Compose generation code (compose.go) — 100% unit tested
- ✅ Plugin metadata types defined
- ✅ Hardened plugin discovery scanner with metadata validation and conflict detection
- ✅ `discoverPluginsInDir` — validates name/version/route, reports typed `DiscoveryIssue`s
- ✅ `LaunchPluginContainers` — extracted with `BuilderClient` interface, fully testable
- ✅ `HTTPBuilderClient` — production builder service client, tested via fake HTTP server
- ✅ Lifecycle state machine: Start / Stop / Restart / Status — full unit test coverage
- ✅ `GET /api/plugins/status` — returns runtime status for all known plugins
- ✅ `POST /api/plugins/{name}/start|stop|restart` — RBAC-gated lifecycle control
- ✅ Plugin health monitoring with HTTP probes and configurable timeouts
- ✅ `GET /api/plugins/diagnostics` — aggregated status + health report
- ✅ All endpoints behind `system.admin` permission guard
- ✅ 97.1% unit test coverage (remaining 2.9% is `ShellComposeRunner.Run` — Docker exec, integration-tagged)
- ✅ `go test ./... -race` — no data races
- ✅ `go vet ./...` — no issues
- ⚠️ `ShellComposeRunner.Run` covered by `lifecycle_integration_test.go` (requires `go test -tags integration`)
- ⚠️ Inter-plugin communication primitives exist (`hooks` + Redis Pub/Sub), but no real plugin traffic yet
- **Remaining**:
  1. Build first reference plugin as proof-of-concept
  2. Run Docker-based static analysis gate (`golangci-lint`, `gosec`, `staticcheck`) in CI

### Completed Recently ✅

**Logging System Integration** (Code: 100%, Integration: 100%)
- ✅ Complete logging service implementation (464 lines)
- ✅ HTTP handlers for all endpoints (398 lines)
- ✅ Database schema with full-text search
- ✅ RBAC-secured endpoints
- ✅ Initialized in main.go
- ✅ Routes registered
- ✅ Logger bridge connected
- ✅ End-to-end log ingestion and search verified with integration tests
- ✅ Logging API unit test suite stabilized and passing across timezone environments

### What's Planned 📋

**Inter-Plugin Communication**
- Current: Redis-backed hooks are implemented and initialized in core; no plugin-to-plugin production flow yet
- Decision still needed: PostgreSQL LISTEN/NOTIFY vs Redis Pub/Sub as long-term event backbone
- Practical state: foundation complete, reliability/security/performance features pending
- Will be finalized with first reference plugins and benchmark/security tests

**TimescaleDB Migration**
- Current: Using regular PostgreSQL
- Logging schema has TimescaleDB optimizations (commented out)
- Plan: Migrate when log volume justifies it (>10K logs/day)

---

## 🗓️ Development Phases

### **Phase 1: Core Infrastructure** (Current - 90% Complete)

**Goal**: Production-ready core services without plugins

**Completed**:
- ✅ RBAC system with all enterprise features
- ✅ Database migrations and schema management
- ✅ Core HTTP server with authentication
- ✅ Builder service for Docker image creation
- ✅ Basic frontend for RBAC testing
- ✅ Docker-first development workflow
- ✅ Logging system integrated into core runtime
- ✅ Plugin discovery hardening with metadata validation and conflict detection
- ✅ Plugin lifecycle management (Start/Stop/Restart/Status) with HTTP API
- ✅ Plugin health monitoring and diagnostics endpoint
- ✅ 97.1% unit test coverage on plugin manager; `go test -race` clean

**Completed Recently**:
- ✅ Builder service image cleanup fix (Docker SDK v27 types, `/cleanup` endpoint live)
- ✅ Reference plugin `asset-tracker` built with TDD (11 tests, 83.5% coverage, race-clean)
- ✅ OpenAPI 3.0.3 documentation at `docs/openapi.yaml`
- ✅ Static analysis gate: `go vet`, `staticcheck`, `gosec` — all clean, 0 HIGH/MEDIUM unaddressed
- ✅ Security hardening: typed context keys, file permission tightening (0750/0640), goroutine context fix

**Remaining**:
- ❌ Inter-plugin communication system
- ❌ CI pipeline integration (Docker-based gate)

**Status**: Phase 1 feature-complete, committed

---

### **Phase 2: Plugin Ecosystem Foundation** (Planned)

**Goal**: Working plugin system with 2-3 reference plugins

**Planned Features**:
- Plugin discovery from /plugins directory
- Docker Compose-based lifecycle management
- Plugin health monitoring and restart logic
- Inter-plugin communication (PostgreSQL LISTEN/NOTIFY or Redis)
- Plugin metadata registry
- Plugin API for accessing core services (RBAC, logging)
- Developer documentation for third-party plugins

**Reference Plugins to Build**:
1. **Locations management**: Building, floor, rooms must be as modular as our RBAC
2. **Asset Management Plugin**: Equipment tracking, basic CRUD operations
3. **Work Order Plugin**: Task creation and assignment
4. **Authentication Plugin**: Extended auth methods (OAuth, LDAP)

**Success Criteria**:
- Core can discover and start plugins automatically
- Plugins can communicate via event system
- Plugins can authenticate and use RBAC
- Plugins can submit logs to central logging system
- Clear developer documentation for plugin creation

**Estimated Duration**: 6-8 weeks

---

### **Phase 3: Production Readiness** (Planned)

**Goal**: Production-grade deployment capabilities

**Features**:
- Comprehensive API documentation (OpenAPI/Swagger)
- Enhanced monitoring and metrics (Prometheus/Grafana)
- TimescaleDB migration for log storage
- Advanced health checks for all services
- Plugin versioning and updates
- Database backup and restore procedures
- Security hardening and penetration testing
- Performance optimization and load testing
- Deployment guides (Docker Compose, Kubernetes-ready architecture)

**Estimated Duration**: 4-6 weeks

---

### **Phase 4: Advanced Features** (Future)

**Goal**: Enterprise-grade features and plugin marketplace

**Planned**:
- Plugin marketplace and discovery system
- Plugin signature verification
- Advanced RBAC features (hierarchical resources, delegation)
- Real-time updates (WebSocket support)
- Multi-region deployment support
- SSO integration (SAML, OAuth2)
- Advanced analytics and reporting
- Mobile app support
- Horizontal scaling and clustering

**Timeline**: TBD based on user feedback

---

## 🎯 Immediate Priorities (Next 2-4 Weeks)

### **Priority 1: Plugin Manager Hardening** ⚡
**Why**: Startup path exists, but operational lifecycle and monitoring are incomplete

**Tasks** (TDD Required):
1. **Write tests first**: Unit tests for discovery/lifecycle/status endpoints
2. Harden plugin discovery scanner (invalid metadata handling, conflict detection)
3. Implement lifecycle operations (start/stop/restart) with deterministic state tracking
4. Add plugin health checks and status monitoring
5. Expand `/api/plugins` endpoints for lifecycle and diagnostics
6. Write security tests for plugin isolation and resource limits
7. Build integration tests for end-to-end lifecycle
8. Run static analysis (golangci-lint, gosec, staticcheck)
9. Run security scans (semgrep, nancy, trivy)
10. Document plugin lifecycle API for developers

**Quality Gates**:
- ✅ All unit tests pass (plugin discovery, lifecycle, monitoring)
- ✅ All integration tests pass (full plugin lifecycle)
- ✅ Security tests verify plugin isolation
- ✅ **Test coverage 100% for core plugin manager code** (matching existing core standard)
- ✅ Zero critical vulnerabilities in plugin system
- ✅ Static analysis passes
- ✅ **Reference plugin has 100% test coverage** (plugins follow core standards)
- ✅ Code review approved

**Estimated**: 2-3 weeks (includes integration/security validation)

---

### **Priority 2: Inter-Plugin Communication** 🔧
**Why**: Hooks foundation exists, but real plugin workflows are not implemented

**Tasks** (TDD Required):
1. Architecture decision: PostgreSQL LISTEN/NOTIFY vs Redis Pub/Sub (with benchmark tests)
2. **Write tests first**: Unit tests for event publishing, subscription, and delivery
3. Implement event publishing API (TDD: failing tests → implementation → passing tests)
4. Implement event subscription for plugins (with integration tests)
5. Add event delivery guarantees with tests (acknowledge, retry, dead-letter)
6. Write security tests for event authentication and authorization
7. Write integration tests with reference plugins
8. Performance testing for event throughput and latency
9. Static analysis on event system code
10. Security scanning for event handling vulnerabilities
11. Document event system for developers

**Quality Gates**:
- ✅ All unit tests pass (publish, subscribe, delivery)
- ✅ All integration tests pass (plugin-to-plugin communication)
- ✅ Performance tests meet targets (<100ms latency, >100 events/sec)
- ✅ Security tests verify event isolation between organizations
- ✅ **Test coverage 100% for core event system** (matching existing core standard)
- ✅ Zero critical security issues
- ✅ Static analysis passes
- ✅ Code review approved

**Estimated**: 2-3 weeks (includes performance and security testing)

---

### **Priority 3: Developer Experience** 📚
**Why**: Enable third-party plugin development

**Tasks**:
1. Create plugin template with Go backend
2. Document plugin development workflow
3. Write plugin API documentation
4. Create example plugins with common patterns
5. Add plugin testing utilities
6. CI/CD pipeline for plugin validation

**Estimated**: 1-2 weeks

---

## 📐 Architectural Decisions

### **Current Architecture**

```
┌─────────────────────────────────────────────────────────────┐
│                     Core Service (Go)                        │
│  ┌─────────────┐  ┌──────────┐  ┌─────────┐  ┌──────────┐ │
│  │ RBAC        │  │ Logging  │  │ Plugin  │  │ Health   │ │
│  │ (Complete)  │  │ (Integrated) │ Manager │  │ (Basic)  │ │
│  └─────────────┘  └──────────┘  └─────────┘  └──────────┘ │
└─────────────────────────────────────────────────────────────┘
                            ▲
                            │ HTTP APIs
                            ▼
┌──────────────┐   ┌─────────────────┐   ┌──────────────┐
│   Builder    │   │   PostgreSQL    │   │    Redis     │
│   Service    │   │   (Active)      │   │ (Hooks Bus   │
│  (90% Done)  │   │ - RBAC Schema   │   │  Foundation) │
│              │   │ - Logging Schema│   │              │
└──────────────┘   └─────────────────┘   └──────────────┘
                            ▲
                            │
                  ┌─────────────────────┐
                  │  Future Plugins     │
                  │  (Not Started)      │
                  └─────────────────────┘
```

### **Target Architecture (Phase 2)**

```
┌─────────────────────────────────────────────────────────────┐
│                    Core Service (Go)                         │
│  ┌─────────────┐  ┌──────────┐  ┌─────────┐  ┌──────────┐ │
│  │ RBAC API    │  │ Logging  │  │ Plugin  │  │ Event    │ │
│  │             │  │ API      │  │ Manager │  │ Bus      │ │
│  └─────────────┘  └──────────┘  └─────────┘  └──────────┘ │
└─────────────────────────────────────────────────────────────┘
         ▲                 ▲               ▲            ▲
         │                 │               │            │
         └─────────────────┴───────────────┴────────────┘
                          HTTP APIs
         ┌─────────────────┴───────────────┬────────────┐
         ▼                 ▼               ▼            ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐ ┌────────┐
│ Asset Plugin │  │  Work Order  │  │    Auth      │ │  ...   │
│ (Container)  │  │   Plugin     │  │   Plugin     │ │        │
└──────────────┘  └──────────────┘  └──────────────┘ └────────┘
         │                 │               │
         └─────────────────┴───────────────┘
                    Event System
              (PostgreSQL LISTEN/NOTIFY
               or Redis Pub/Sub)
```

### **Key Decisions**

**Database**: PostgreSQL
- ✅ Decided: PostgreSQL for core data
- ✅ Multi-tenant with organization_id
- 🔄 Pending: TimescaleDB migration for high-volume logs
- ✅ Plugins can use sidecar databases (Elasticsearch, etc.)

**Plugin Communication**: TBD
- 🤔 Option A: PostgreSQL LISTEN/NOTIFY (simpler, integrated)
- 🤔 Option B: Redis Pub/Sub (faster, scalable)
- 📊 Recommendation: Start with PostgreSQL, migrate to Redis if needed
- ⏰ Decision deadline: End of Phase 1

**Orchestration**: Docker Compose → Kubernetes-ready
- ✅ Current: Docker Compose for development
- ✅ Architecture designed for K8s migration
- 📋 Future: Support both Docker Compose and Kubernetes

---

## ✅ Completed Milestones

### **December 2024: RBAC Foundation**
- ✅ Complete RBAC system with PostgreSQL backend
- ✅ Time-based access control
- ✅ Emergency access workflows with break-glass
- ✅ JWT authentication with organization context
- ✅ Audit middleware and comprehensive logging
- ✅ Migration system with auto-apply
- ✅ Background processors for scheduled access

### **November 2024: Project Foundation**
- ✅ Docker-first development setup
- ✅ Core service HTTP server with health checks
- ✅ Builder service with API key auth
- ✅ Configuration management via environment variables
- ✅ Logging infrastructure (console + file)
- ✅ Database connection pooling
- ✅ Graceful shutdown handling

### **October 2024: Initial Setup**
- ✅ Repository structure and Go modules
- ✅ Docker Compose configuration
- ✅ Development guidelines documentation
- ✅ Basic CI/CD workflows
- ✅ Code cleanup and organization

---

## 📚 Documentation Status

### **Complete & Current**
- ✅ [GUIDELINES.md](GUIDELINES.md) - Development standards and principles
- ✅ [docs/RBAC_ARCHITECTURE.md](docs/RBAC_ARCHITECTURE.md) - RBAC system design
- ✅ [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) - System architecture overview
- ✅ [CLEANUP_SUMMARY.md](CLEANUP_SUMMARY.md) - Recent cleanup work

### **Design Docs (Implementation Pending)**
- 📋 [docs/LOGGING_SYSTEM_DESIGN.md](docs/LOGGING_SYSTEM_DESIGN.md) - Logging architecture (implemented; doc refresh needed)
- 📋 [docs/PLUGIN_SIDECAR_PATTERN.md](docs/PLUGIN_SIDECAR_PATTERN.md) - Plugin database strategy (planned)
- 📋 [docs/EXTERNAL_PLUGIN_LOGGING.md](docs/EXTERNAL_PLUGIN_LOGGING.md) - Plugin logging integration (planned)

### **Needs Creation**
- ❌ API Documentation (OpenAPI/Swagger)
- ❌ Plugin Development Guide
- ❌ Deployment Guide
- ❌ Security Best Practices
- ❌ Performance Tuning Guide

---

## 🚧 Known Issues & Technical Debt

> **Technical Debt Policy**: Zero tolerance for new technical debt. All known issues must have tests and remediation plans.

### **High Priority**
1. **Plugin Lifecycle Controls Incomplete**: Startup path exists but robust lifecycle/status APIs are missing
   - **Tests needed**: Full TDD implementation with unit + integration tests
   - **Security**: Must pass plugin isolation security tests
2. **Hooks Not Exercised by Real Plugins**: Event primitives exist but no plugin-to-plugin production flow
   - **Tests needed**: Performance benchmarks for PostgreSQL LISTEN/NOTIFY vs Redis
   - **Decision**: Based on test results, not assumptions
3. **Builder Image Cleanup**: Disabled due to Docker API type compatibility
   - **Tests needed**: Unit tests for image cleanup logic
   - **Fix**: Must include integration tests before re-enabling

### **Medium Priority**
1. **TimescaleDB Migration**: Schema ready but using regular PostgreSQL
   - **Tests needed**: Performance benchmarks to justify migration
   - **Quality gate**: Only migrate when load testing proves need
2. **API Documentation**: No OpenAPI/Swagger specs
   - **Tests needed**: API contract tests to validate documentation accuracy
3. **Test Coverage Gaps**: Some older utility code may need review
   - **Action**: Achieve and maintain 100% coverage for all core code (current standard)
4. **Error Handling**: Some error paths need better handling
   - **Tests needed**: Error path testing for all critical flows

### **Low Priority (But Must Have Tests)**
1. **Frontend Polish**: Demo only, needs redesign
   - **Tests needed**: Frontend unit tests + E2E tests before production
2. **Metrics/Monitoring**: Basic health checks only
   - **Tests needed**: Monitoring endpoint tests
3. **Rate Limiting**: Not implemented
   - **Tests needed**: Load tests to prove rate limiting works
4. **Caching Layer**: No caching strategy yet
   - **Tests needed**: Performance tests to justify caching

### **Test Coverage Status**
- **Current Core Packages**: 100% coverage achieved ✅
- **Standard for All Core Code**: 100% coverage required (strictly enforced)
- **Plugin Code**: 100% coverage required (plugins follow core standards)
- **New Code**: 100% coverage required before merge (TDD enforced)
- **No Exceptions**: Every line of core and plugin code must be tested

---

## 🎓 Lessons Learned

### **What Worked Well**
- ✅ Docker-first development is very effective
- ✅ RBAC implementation is comprehensive and extensible
- ✅ Migration system prevents schema drift
- ✅ Structured logging helps debugging
- ✅ Clear separation: builder service isolated from core
- ✅ Test-driven development ensures code quality from the start

### **What to Improve**
- ⚠️ Build one complete feature before starting next
- ⚠️ Keep documentation in sync with implementation
- ⚠️ Start with simple plugin, then generalize
- ⚠️ Don't add services (Redis) until actually needed
- ⚠️ **Always write tests BEFORE implementation** (enforce TDD strictly)
- ⚠️ Run security scans continuously, not just before releases

### **Course Corrections**
- 🔄 Build reference plugin before generalizing plugin system
- 🔄 Decide on event system (PostgreSQL vs Redis) based on actual needs
- 🔄 Document APIs as they're built, not after
- 🔄 **Mandatory TDD**: No code merges without tests written first
- 🔄 **Continuous security**: Run static analysis and security scans on every feature branch
- 🔄 **Quality gates**: Enforce 100% test coverage, zero critical vulnerabilities before merge

---

## 💡 Contributing

This is an early-stage project. Priority areas for contribution:

1. **Plugin Lifecycle**: Complete start/stop/restart/status and health monitoring
2. **Inter-Plugin Events**: Implement and validate real plugin event workflows
3. **Reference Plugins**: Build example plugins to prove orchestration + communication
4. **Documentation**: API docs, deployment guides, plugin development guides
5. **Testing**: Integration tests, load tests, security tests

For questions or to discuss priorities, please open an issue!

---

**Last Updated**: March 27, 2026
**Project Status**: Early Development - Core Infrastructure Phase
**Next Milestone**: Plugin lifecycle hardening + inter-plugin event MVP
