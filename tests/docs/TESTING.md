# Maintify Testing Infrastructure

## 🐳 100% Docker-Based Testing

**No Local Installation Required!** All testing runs in Docker containers - you only need Docker and docker-compose installed.

### ⚡ Quick Start (Docker-Only)

```bash
# Option 1: Using Make (recommended)
make test                 # Complete test suite
make test-quick          # Fast feedback (skip security)
make test-dashboard      # View results at http://localhost:8080

# Option 2: Direct Docker Compose
docker-compose -f docker-compose.test.yml run --rm unit-tests
docker-compose -f docker-compose.test.yml run --rm integration-tests
```

### 🎯 Individual Test Types (All Docker-Based)

```bash
# Unit tests only (fastest)
make test-unit

# Integration tests with real database
make test-integration

# Static analysis and code quality
make test-static

# Security vulnerability scans
make test-security

# Quick development feedback
make test-quick
```

### 🧹 Test Environment Management

```bash
# Clean up everything
make test-clean

# Build test containers (after Dockerfile changes)
make test-build

# View test environment status
docker-compose -f docker-compose.test.yml ps
```

## 🏗️ Testing Architecture

### Docker Compose Services
- **test-db**: Isolated PostgreSQL for testing
- **test-redis**: Isolated Redis for testing
- **unit-tests**: Go unit test runner
- **integration-tests**: Go integration test runner
- **static-analysis**: Code quality and lint analysis
- **security-scan**: Vulnerability and security scanning
- **sonarqube**: Code quality dashboard
- **test-dashboard**: HTML results dashboard

### Test Results Structure
```
test-results/
├── unit-tests/
│   ├── coverage.html       # Unit test coverage report
│   ├── coverage.out        # Go coverage data
│   ├── coverage.txt        # Coverage percentage
│   └── junit.xml          # JUnit test results
├── integration-tests/
│   ├── coverage.html       # Integration test coverage
│   ├── coverage.out        # Go coverage data
│   ├── coverage.txt        # Coverage percentage
│   └── junit.xml          # JUnit test results
├── static-analysis/
│   ├── summary.html        # Analysis summary
│   ├── golangci-lint.xml   # Linting results
│   ├── gosec.json         # Security analysis
│   ├── gocyclo.txt        # Complexity analysis
│   └── staticcheck.xml    # Static check results
├── security/
│   ├── summary.html        # Security summary
│   ├── nancy-dependencies.txt # Dependency vulnerabilities
│   ├── semgrep-security.json  # Code security patterns
│   └── trivy-summary.txt      # Container vulnerabilities
└── index.html             # Main dashboard
```

## 🔧 Configuration

### GolangCI-Lint
Configuration in `.golangci.yml` with enabled linters:
- errcheck, gosec, staticcheck, ineffassign
- gocyclo, govet, gofmt, goimports
- Custom rules for test files and error handling

### SonarQube
Configuration in `sonar-project.properties`:
- Go coverage integration
- Quality gate requirements
- Security hotspot detection

## 🚀 CI/CD Integration

### GitHub Actions Example
```yaml
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Run Tests
        run: ./scripts/run-all-tests.sh
      - name: Upload Results
        uses: actions/upload-artifact@v3
        with:
          name: test-results
          path: test-results/
```

### GitLab CI Example
```yaml
test:
  stage: test
  script:
    - ./scripts/run-all-tests.sh
  artifacts:
    reports:
      junit: test-results/*/junit.xml
    paths:
      - test-results/
```

## 📊 Quality Gates

### Required Standards
- **Unit Test Coverage**: ≥80%
- **Integration Test Coverage**: All API endpoints
- **Static Analysis**: Zero critical issues
- **Security Scan**: Zero critical vulnerabilities
- **Performance**: No API response time regression

### Dashboard Access
- **Local Dashboard**: http://localhost:8080
- **SonarQube**: http://localhost:9000
- **Test Coverage**: Click through from dashboard

## 🛠️ Troubleshooting

### Common Issues
```bash
# Permission issues
chmod +x scripts/*.sh

# Docker permission issues
sudo usermod -a -G docker $USER

# Port conflicts
docker-compose -f docker-compose.test.yml down

# Clean test environment
docker system prune -f
```

### Debug Mode
```bash
# Run with verbose output
docker-compose -f docker-compose.test.yml run --rm unit-tests /bin/bash

# Check test logs
docker-compose -f docker-compose.test.yml logs unit-tests
```