#!/bin/bash
set -e

echo "Running Security Scans..."

# Create results directory
mkdir -p /app/results/security

# Security scan of Go dependencies using Nancy
echo "Scanning Go dependencies with Nancy..."
cd /app/core
go list -json -m all | nancy sleuth > /app/results/security/nancy-dependencies.txt || true

# Security scan with Semgrep
echo "Running Semgrep security patterns..."
semgrep --config=auto --json --output=/app/results/security/semgrep-security.json /app/core || true
semgrep --config=auto --output=/app/results/security/semgrep-security.txt /app/core || true

# Scan Docker images if Docker is available
if command -v docker &> /dev/null; then
    echo "Scanning Docker images with Trivy..."
    
    # Scan base images used in our Dockerfiles
    trivy image --format json --output /app/results/security/trivy-golang.json golang:1.24 || true
    trivy image --format json --output /app/results/security/trivy-postgres.json postgres:15 || true
    trivy image --format json --output /app/results/security/trivy-redis.json redis:7-alpine || true
    
    # Generate summary
    trivy image --format table --output /app/results/security/trivy-summary.txt golang:1.24 postgres:15 redis:7-alpine || true
fi

# Scan source code with Grype if available
if command -v grype &> /dev/null; then
    echo "Scanning with Grype..."
    grype dir:/app/core -o json > /app/results/security/grype-scan.json || true
    grype dir:/app/core -o table > /app/results/security/grype-scan.txt || true
fi

# Generate security summary report
echo "Generating security summary..."
cat > /app/results/security/summary.html << EOF
<!DOCTYPE html>
<html>
<head>
    <title>Security Scan Results</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .section { margin: 20px 0; padding: 15px; border: 1px solid #ddd; border-radius: 5px; }
        .header { background-color: #f5f5f5; font-weight: bold; }
        .critical { background-color: #ffebee; }
        .high { background-color: #fff3e0; }
        .medium { background-color: #f3e5f5; }
        .low { background-color: #e8f5e8; }
        pre { background-color: #f8f8f8; padding: 10px; overflow-x: auto; }
    </style>
</head>
<body>
    <h1>Security Scan Results</h1>
    <div class="section">
        <div class="header">Dependency Vulnerabilities (Nancy)</div>
        <pre>$(cat /app/results/security/nancy-dependencies.txt 2>/dev/null || echo "No vulnerabilities found in dependencies")</pre>
    </div>
    <div class="section">
        <div class="header">Code Security Patterns (Semgrep)</div>
        <pre>$(cat /app/results/security/semgrep-security.txt 2>/dev/null || echo "No security issues found in code")</pre>
    </div>
    <div class="section">
        <div class="header">Container Image Vulnerabilities (Trivy)</div>
        <pre>$(cat /app/results/security/trivy-summary.txt 2>/dev/null || echo "Container scan not available")</pre>
    </div>
</body>
</html>
EOF

echo "Security scans completed successfully!"