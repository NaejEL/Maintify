#!/bin/bash
set -e

echo "Starting Maintify Test Suite"
echo "================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Create test results directory
mkdir -p test-results

# Function to run a specific test type
run_test() {
    local test_type=$1
    local service_name=$2
    
    print_status "Running $test_type..."
    
    if docker-compose -f docker-compose.test.yml run --rm $service_name; then
        print_success "$test_type completed successfully"
        return 0
    else
        print_error "$test_type failed"
        return 1
    fi
}

# Parse command line arguments
RUN_UNIT=true
RUN_INTEGRATION=true
RUN_STATIC=true
RUN_SECURITY=true
QUICK_MODE=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --unit-only)
            RUN_INTEGRATION=false
            RUN_STATIC=false
            RUN_SECURITY=false
            shift
            ;;
        --integration-only)
            RUN_UNIT=false
            RUN_STATIC=false
            RUN_SECURITY=false
            shift
            ;;
        --static-only)
            RUN_UNIT=false
            RUN_INTEGRATION=false
            RUN_SECURITY=false
            shift
            ;;
        --security-only)
            RUN_UNIT=false
            RUN_INTEGRATION=false
            RUN_STATIC=false
            shift
            ;;
        --quick)
            QUICK_MODE=true
            RUN_SECURITY=false
            shift
            ;;
        --help)
            echo "Usage: $0 [options]"
            echo "Options:"
            echo "  --unit-only       Run only unit tests"
            echo "  --integration-only Run only integration tests"
            echo "  --static-only     Run only static analysis"
            echo "  --security-only   Run only security scans"
            echo "  --quick           Skip security scans for faster execution"
            echo "  --help            Show this help message"
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Ensure Docker Compose is available
if ! command -v docker-compose &> /dev/null; then
    print_error "docker-compose is required but not installed"
    exit 1
fi

# Start required services
print_status "Starting test infrastructure..."
docker-compose -f docker-compose.test.yml up -d test-db test-redis sonarqube

# Wait for services to be ready
print_status "Waiting for services to be ready..."
sleep 10

# Track test results
FAILED_TESTS=()

# Run tests based on flags
if [ "$RUN_UNIT" = true ]; then
    if ! run_test "Unit Tests" "unit-tests"; then
        FAILED_TESTS+=("Unit Tests")
    fi
fi

if [ "$RUN_INTEGRATION" = true ]; then
    if ! run_test "Integration Tests" "integration-tests"; then
        FAILED_TESTS+=("Integration Tests")
    fi
fi

if [ "$RUN_STATIC" = true ]; then
    if ! run_test "Static Analysis" "static-analysis"; then
        FAILED_TESTS+=("Static Analysis")
    fi
fi

if [ "$RUN_SECURITY" = true ]; then
    if ! run_test "Security Scans" "security-scan"; then
        FAILED_TESTS+=("Security Scans")
    fi
fi

# Generate combined test report
print_status "Generating test dashboard..."
docker-compose -f docker-compose.test.yml up -d test-dashboard

# Create main dashboard index
cat > test-results/index.html << 'EOF'
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Maintify Test Results Dashboard</title>
    <style>
        body { 
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; 
            margin: 0; 
            padding: 20px; 
            background-color: #f5f5f5; 
        }
        .container { max-width: 1200px; margin: 0 auto; }
        .header { 
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); 
            color: white; 
            padding: 2rem; 
            border-radius: 10px; 
            margin-bottom: 2rem; 
            text-align: center; 
        }
        .grid { 
            display: grid; 
            grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); 
            gap: 20px; 
            margin-bottom: 2rem; 
        }
        .card { 
            background: white; 
            border-radius: 10px; 
            padding: 1.5rem; 
            box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1); 
            transition: transform 0.2s; 
        }
        .card:hover { transform: translateY(-2px); }
        .card h3 { 
            margin-top: 0; 
            color: #333; 
            border-bottom: 2px solid #eee; 
            padding-bottom: 0.5rem; 
        }
        .status { 
            display: inline-block; 
            padding: 0.25rem 0.75rem; 
            border-radius: 15px; 
            font-size: 0.875rem; 
            font-weight: 600; 
        }
        .status.success { background-color: #d4edda; color: #155724; }
        .status.warning { background-color: #fff3cd; color: #856404; }
        .status.error { background-color: #f8d7da; color: #721c24; }
        .links { list-style: none; padding: 0; }
        .links li { margin: 0.5rem 0; }
        .links a { 
            color: #667eea; 
            text-decoration: none; 
            font-weight: 500; 
        }
        .links a:hover { text-decoration: underline; }
        .timestamp { 
            color: #666; 
            font-size: 0.875rem; 
            text-align: center; 
            margin-top: 2rem; 
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Maintify Test Results Dashboard</h1>
            <p>Comprehensive testing and quality analysis results</p>
        </div>
        
        <div class="grid">
            <div class="card">
                <h3>Unit Tests</h3>
                <span class="status success">Available</span>
                <ul class="links">
                    <li><a href="unit-tests/coverage.html">Coverage Report</a></li>
                    <li><a href="unit-tests/junit.xml">JUnit Results</a></li>
                    <li><a href="unit-tests/coverage.txt">Coverage %</a></li>
                </ul>
            </div>
            
            <div class="card">
                <h3>Integration Tests</h3>
                <span class="status success">Available</span>
                <ul class="links">
                    <li><a href="integration-tests/coverage.html">Coverage Report</a></li>
                    <li><a href="integration-tests/junit.xml">JUnit Results</a></li>
                    <li><a href="integration-tests/coverage.txt">Coverage %</a></li>
                </ul>
            </div>
            
            <div class="card">
                <h3>Static Analysis</h3>
                <span class="status warning">Review Required</span>
                <ul class="links">
                    <li><a href="static-analysis/summary.html">Analysis Summary</a></li>
                    <li><a href="static-analysis/golangci-lint.xml">GolangCI-Lint</a></li>
                    <li><a href="static-analysis/gosec.json">Security Analysis</a></li>
                    <li><a href="static-analysis/gocyclo.txt">Complexity Report</a></li>
                </ul>
            </div>
            
            <div class="card">
                <h3>Security Scans</h3>
                <span class="status error">Issues Found</span>
                <ul class="links">
                    <li><a href="security/summary.html">Security Summary</a></li>
                    <li><a href="security/nancy-dependencies.txt">Dependency Scan</a></li>
                    <li><a href="security/semgrep-security.json">Code Security</a></li>
                    <li><a href="security/trivy-summary.txt">Container Scan</a></li>
                </ul>
            </div>
        </div>
        
        <div class="timestamp">
            Last updated: <span id="timestamp"></span>
        </div>
    </div>
    
    <script>
        document.getElementById('timestamp').textContent = new Date().toLocaleString();
    </script>
</body>
</html>
EOF

# Cleanup
print_status "Cleaning up test infrastructure..."
docker-compose -f docker-compose.test.yml down

# Print final results
echo ""
echo "================================"
if [ ${#FAILED_TESTS[@]} -eq 0 ]; then
    print_success "All tests passed!"
    echo ""
    print_status "View results at: http://localhost:8080"
    echo ""
    exit 0
else
    print_error "Some tests failed:"
    for test in "${FAILED_TESTS[@]}"; do
        echo "  - $test"
    done
    echo ""
    print_status "View detailed results at: http://localhost:8080"
    echo ""
    exit 1
fi