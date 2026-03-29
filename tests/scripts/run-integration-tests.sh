#!/bin/bash
set -e

echo "Running Integration Tests..."

# Change to core directory
cd /app/core

# Create results directory
mkdir -p /app/results/integration-tests

# Wait for database to be ready
echo "Waiting for test database..."
until pg_isready -h $DB_HOST -p $DB_PORT -U $DB_USER; do
    echo "Waiting for database..."
    sleep 2
done

# Wait for Redis to be ready
echo "Waiting for test Redis..."
until redis-cli -h $REDIS_HOST -p $REDIS_PORT ping; do
    echo "Waiting for Redis..."
    sleep 2
done

# Run database migrations
echo "Running database migrations..."
go run cmd/rbac-cli/main.go migrate

# Download dependencies
echo "Downloading dependencies..."
go mod download

# Packages excluded from testing (Phase 2 placeholders — no tests yet):
#   pkg/plugin   — Plugin interface/base definitions, wired up in Phase 2
#   pkg/registry — Simple plugin registry stub, wired up in Phase 2
EXCLUDED_PKGS="maintify/core/pkg/plugin$|maintify/core/pkg/registry$"
PACKAGES=$(go list ./... | grep -Ev "$EXCLUDED_PKGS")

# Run integration tests
echo "Running integration tests..."
gotestsum --format testname --junitfile /app/results/integration-tests/junit.xml -- \
    -tags=integration \
    -coverprofile=/app/results/integration-tests/coverage.out \
    -covermode=atomic \
    -timeout=30m \
    $PACKAGES

# Generate coverage report
echo "Generating integration coverage report..."
go tool cover -html=/app/results/integration-tests/coverage.out -o /app/results/integration-tests/coverage.html

# Calculate coverage percentage
COVERAGE=$(go tool cover -func=/app/results/integration-tests/coverage.out | grep total | awk '{print $3}')
echo "Integration coverage: $COVERAGE"

# Save coverage percentage to file
echo "$COVERAGE" > /app/results/integration-tests/coverage.txt

echo "Integration tests completed successfully!"