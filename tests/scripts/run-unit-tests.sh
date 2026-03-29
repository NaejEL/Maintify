#!/bin/bash
set -e

echo "Running Unit Tests..."

# Change to core directory
cd /app/core

# Create results directory
mkdir -p /app/results/unit-tests

# Download dependencies
echo "Downloading dependencies..."
go mod download

# Packages excluded from testing (Phase 2 placeholders — no tests yet):
#   pkg/plugin   — Plugin interface/base definitions, wired up in Phase 2
#   pkg/registry — Simple plugin registry stub, wired up in Phase 2
EXCLUDED_PKGS="maintify/core/pkg/plugin$|maintify/core/pkg/registry$"

# Build the list of testable packages by excluding Phase 2 stubs
PACKAGES=$(go list ./... | grep -Ev "$EXCLUDED_PKGS")

# Run unit tests with coverage
echo "Running unit tests..."
gotestsum --format testname --junitfile /app/results/unit-tests/junit.xml -- \
    -coverprofile=/app/results/unit-tests/coverage.out \
    -covermode=atomic \
    -race \
    -timeout=10m \
    $PACKAGES

# Generate coverage report
echo "Generating coverage report..."
go tool cover -html=/app/results/unit-tests/coverage.out -o /app/results/unit-tests/coverage.html

# Calculate coverage percentage
COVERAGE=$(go tool cover -func=/app/results/unit-tests/coverage.out | grep total | awk '{print $3}')
echo "Total coverage: $COVERAGE"

# Save coverage percentage to file
echo "$COVERAGE" > /app/results/unit-tests/coverage.txt

echo "Unit tests completed successfully!"
