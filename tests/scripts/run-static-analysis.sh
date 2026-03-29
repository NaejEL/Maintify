#!/bin/bash
set -e

echo "Running Static Analysis..."

# Change to core directory
cd /app/core

# Create results directory
mkdir -p /app/results/static-analysis

# Download dependencies
echo "Downloading dependencies..."
go mod download

# Run golangci-lint
echo "Running golangci-lint..."
golangci-lint run --out-format checkstyle > /app/results/static-analysis/golangci-lint.xml || true
golangci-lint run --out-format tab > /app/results/static-analysis/golangci-lint.txt || true

# Run gosec security analysis
echo "Running gosec security analysis..."
gosec -fmt json -out /app/results/static-analysis/gosec.json ./... || true
gosec -fmt text -out /app/results/static-analysis/gosec.txt ./... || true

# Run gocyclo complexity analysis
echo "Running cyclomatic complexity analysis..."
gocyclo -over 10 . > /app/results/static-analysis/gocyclo.txt || true

# Run ineffassign
echo "Running ineffassign analysis..."
ineffassign ./... > /app/results/static-analysis/ineffassign.txt || true

# Run misspell
echo "Running misspell check..."
misspell -error . > /app/results/static-analysis/misspell.txt || true

# Run staticcheck
echo "Running staticcheck..."
staticcheck -f checkstyle ./... > /app/results/static-analysis/staticcheck.xml || true
staticcheck ./... > /app/results/static-analysis/staticcheck.txt || true

# Run errcheck
echo "Running errcheck..."
errcheck ./... > /app/results/static-analysis/errcheck.txt || true

# Run semgrep if available
if command -v semgrep &> /dev/null; then
    echo "Running semgrep security patterns..."
    semgrep --config=auto --json --output=/app/results/static-analysis/semgrep.json . || true
fi

# Generate summary report
echo "Generating static analysis summary..."
cat > /app/results/static-analysis/summary.html << EOF
<!DOCTYPE html>
<html>
<head>
    <title>Static Analysis Results</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .section { margin: 20px 0; padding: 15px; border: 1px solid #ddd; border-radius: 5px; }
        .header { background-color: #f5f5f5; font-weight: bold; }
        pre { background-color: #f8f8f8; padding: 10px; overflow-x: auto; }
    </style>
</head>
<body>
    <h1>Static Analysis Results</h1>
    <div class="section">
        <div class="header">GolangCI-Lint Results</div>
        <pre>$(cat /app/results/static-analysis/golangci-lint.txt 2>/dev/null || echo "No issues found")</pre>
    </div>
    <div class="section">
        <div class="header">Security Analysis (gosec)</div>
        <pre>$(cat /app/results/static-analysis/gosec.txt 2>/dev/null || echo "No security issues found")</pre>
    </div>
    <div class="section">
        <div class="header">Cyclomatic Complexity</div>
        <pre>$(cat /app/results/static-analysis/gocyclo.txt 2>/dev/null || echo "No complex functions found")</pre>
    </div>
</body>
</html>
EOF

echo "Static analysis completed successfully!"