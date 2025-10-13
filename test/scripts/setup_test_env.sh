#!/bin/bash

# TUI Test Environment Setup Script
# Configures the testing environment for PTY-based TUI testing
# This script ensures all dependencies, configurations, and test data are properly set up

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
WHITE='\033[1;37m'
NC='\033[0m' # No Color

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
TEST_DATA_DIR="$PROJECT_ROOT/test/data"
TEST_CONFIG_DIR="$PROJECT_ROOT/test/config"
TEST_FIXTURES_DIR="$PROJECT_ROOT/test/fixtures"

# Setup options
VERBOSE="${VERBOSE:-false}"
FORCE_REBUILD="${FORCE_REBUILD:-false}"
SKIP_DEPS="${SKIP_DEPS:-false}"
CREATE_FIXTURES="${CREATE_FIXTURES:-true}"
SETUP_CI="${SETUP_CI:-false}"
DRY_RUN="${DRY_RUN:-false}"

# Utility functions
log() {
    echo -e "${BLUE}[SETUP] $1${NC}"
}

log_success() {
    echo -e "${GREEN}[SETUP] ✓ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}[SETUP] ⚠ $1${NC}"
}

log_error() {
    echo -e "${RED}[SETUP] ✗ $1${NC}"
}

log_info() {
    if [[ "$VERBOSE" == "true" ]]; then
        echo -e "${WHITE}[SETUP] ℹ $1${NC}"
    fi
}

show_usage() {
    cat << EOF
TUI Test Environment Setup Script

USAGE:
    $0 [OPTIONS]

OPTIONS:
    -v, --verbose           Enable verbose output
    -f, --force-rebuild     Force rebuild of binary and dependencies
    -s, --skip-deps         Skip dependency installation
    -n, --no-fixtures       Skip test fixture creation
    -c, --setup-ci          Setup for CI/CD environment
    -d, --dry-run           Show what would be done without executing
    -h, --help             Show this help message

EXAMPLES:
    $0                      # Standard setup
    $0 -v -f               # Verbose setup with forced rebuild
    $0 -c                  # Setup for CI/CD environment
    $0 --dry-run           # Preview setup actions

ENVIRONMENT VARIABLES:
    VERBOSE                Enable verbose output (true/false)
    FORCE_REBUILD          Force rebuild (true/false)
    SKIP_DEPS              Skip dependencies (true/false)
    CREATE_FIXTURES        Create test fixtures (true/false)
    SETUP_CI               CI/CD setup (true/false)
    DRY_RUN                Dry run mode (true/false)

SETUP ACTIONS:
    1. Create directory structure
    2. Validate Go environment
    3. Install dependencies
    4. Build brummer binary
    5. Create test configuration
    6. Generate test fixtures
    7. Setup test isolation
    8. Validate PTY availability
    9. Create helper scripts

EOF
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -v|--verbose)
                VERBOSE="true"
                shift
                ;;
            -f|--force-rebuild)
                FORCE_REBUILD="true"
                shift
                ;;
            -s|--skip-deps)
                SKIP_DEPS="true"
                shift
                ;;
            -n|--no-fixtures)
                CREATE_FIXTURES="false"
                shift
                ;;
            -c|--setup-ci)
                SETUP_CI="true"
                shift
                ;;
            -d|--dry-run)
                DRY_RUN="true"
                shift
                ;;
            -h|--help)
                show_usage
                exit 0
                ;;
            *)
                log_error "Unknown argument: $1"
                show_usage
                exit 2
                ;;
        esac
    done
}

# Execute command or show what would be executed
execute_or_dry_run() {
    local description="$1"
    shift

    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY RUN] Would execute: $*"
        log_info "[DRY RUN] Description: $description"
        return 0
    else
        log "$description"
        "$@"
    fi
}

# Create directory structure
create_directories() {
    log "Creating test directory structure..."

    local directories=(
        "$TEST_DATA_DIR"
        "$TEST_CONFIG_DIR"
        "$TEST_FIXTURES_DIR"
        "$PROJECT_ROOT/test/results"
        "$PROJECT_ROOT/test/logs"
        "$PROJECT_ROOT/test/logs/validation"
        "$PROJECT_ROOT/test/tmp"
        "$PROJECT_ROOT/test/workspace"
    )

    for dir in "${directories[@]}"; do
        if [[ ! -d "$dir" ]]; then
            execute_or_dry_run "Creating directory: $dir" mkdir -p "$dir"
            if [[ "$DRY_RUN" != "true" && -d "$dir" ]]; then
                log_success "Created directory: $dir"
            fi
        else
            log_info "Directory already exists: $dir"
        fi
    done

    # Create .gitkeep files to preserve empty directories
    local keep_dirs=(
        "$TEST_DATA_DIR"
        "$TEST_CONFIG_DIR"
        "$TEST_FIXTURES_DIR"
        "$PROJECT_ROOT/test/results"
        "$PROJECT_ROOT/test/logs"
        "$PROJECT_ROOT/test/tmp"
        "$PROJECT_ROOT/test/workspace"
    )

    for dir in "${keep_dirs[@]}"; do
        local keep_file="$dir/.gitkeep"
        if [[ ! -f "$keep_file" ]]; then
            execute_or_dry_run "Creating .gitkeep file: $keep_file" touch "$keep_file"
        fi
    done
}

# Validate Go environment
validate_go_environment() {
    log "Validating Go environment..."

    # Check Go installation
    if ! command -v go > /dev/null 2>&1; then
        log_error "Go is not installed or not in PATH"
        log_info "Please install Go from https://golang.org/dl/"
        return 1
    fi

    local go_version=$(go version 2>/dev/null || echo "Unknown")
    log_success "Go version: $go_version"

    # Check Go modules
    if [[ ! -f "$PROJECT_ROOT/go.mod" ]]; then
        log_error "go.mod file not found in project root"
        return 1
    fi

    # Check minimum Go version (1.21+ recommended)
    local go_major=$(go version | awk '{print $3}' | sed 's/go//' | cut -d. -f1)
    local go_minor=$(go version | awk '{print $3}' | sed 's/go//' | cut -d. -f2)

    if [[ $go_major -lt 1 || ($go_major -eq 1 && $go_minor -lt 21) ]]; then
        log_warning "Go version < 1.21 detected. Some features may not work correctly."
    fi

    # Validate GOPATH and GOCACHE
    local gopath=${GOPATH:-$(go env GOPATH)}
    local gocache=${GOCACHE:-$(go env GOCACHE)}

    log_info "GOPATH: $gopath"
    log_info "GOCACHE: $gocache"

    # Check for sufficient disk space
    if command -v df > /dev/null 2>&1; then
        local available_space=$(df -h "$PROJECT_ROOT" | awk 'NR==2 {print $4}')
        log_info "Available disk space: $available_space"
    fi

    log_success "Go environment validation completed"
}

# Install dependencies
install_dependencies() {
    if [[ "$SKIP_DEPS" == "true" ]]; then
        log_info "Skipping dependency installation"
        return 0
    fi

    log "Installing Go dependencies..."

    # Download dependencies
    execute_or_dry_run "Downloading Go modules" go mod download

    # Verify dependencies
    execute_or_dry_run "Verifying Go modules" go mod verify

    # Tidy dependencies
    execute_or_dry_run "Tidying Go modules" go mod tidy

    # Check for missing dependencies
    if ! go mod tidy > /dev/null 2>&1; then
        log_error "Failed to tidy Go modules"
        return 1
    fi

    # Install common testing tools
    local test_tools=(
        "github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
        "gotest.tools/gotestsum@latest"
    )

    for tool in "${test_tools[@]}"; do
        local tool_name=$(echo "$tool" | cut -d'@' -f1 | rev | cut -d'/' -f1 | rev)
        if ! command -v "$tool_name" > /dev/null 2>&1; then
            log_info "Installing $tool_name..."
            execute_or_dry_run "Installing $tool_name" go install "$tool"
        else
            log_info "$tool_name already installed"
        fi
    done

    # Check for PTY dependencies
    log_info "Checking PTY dependencies..."
    if ! go list -m github.com/creack/pty > /dev/null 2>&1; then
        log_error "PTY dependency (github.com/creack/pty) not found"
        log_info "Adding PTY dependency..."
        execute_or_dry_run "Adding PTY dependency" go get github.com/creack/pty@v1.1.21
    else
        log_success "PTY dependency found"
    fi

    log_success "Dependency installation completed"
}

# Build brummer binary
build_binary() {
    log "Building brummer binary..."

    local binary_path="$PROJECT_ROOT/brum"
    local build_args=()

    if [[ "$FORCE_REBUILD" == "true" || ! -f "$binary_path" ]]; then
        log_info "Building brummer binary..."

        # Add build tags for testing
        build_args+=("-tags=tty,integration")

        # Set build time variables
        build_args+=("-ldflags" "-X main.version=test-setup -X main.buildTime=$(date -Iseconds)")

        # Build the binary
        execute_or_dry_run "Building brummer binary" go build "${build_args[@]}" -o "$binary_path" ./cmd/brum

        if [[ "$DRY_RUN" != "true" && -f "$binary_path" ]]; then
            # Make binary executable
            chmod +x "$binary_path"
            log_success "Built brummer binary: $binary_path"

            # Test binary functionality
            if timeout 5s "$binary_path" --no-tui --help > /dev/null 2>&1; then
                log_success "Binary functionality verified"
            else
                log_warning "Binary functionality test failed"
            fi
        fi
    else
        log_info "Binary already exists, skipping build (use --force-rebuild to rebuild)"
    fi
}

# Create test configuration
create_test_configuration() {
    log "Creating test configuration..."

    # Create test environment configuration
    local test_config="$TEST_CONFIG_DIR/test_env.toml"

    if [[ ! -f "$test_config" || "$FORCE_REBUILD" == "true" ]]; then
        cat > "$test_config" << 'EOF'
# TUI Test Environment Configuration
[environment]
# Test environment settings
test_mode = true
verbose_logging = false
pty_required = true
timeout_seconds = 30

[testing]
# Testing-specific settings
enable_debug = false
capture_logs = true
keep_test_data = false
parallel_tests = false

[pty]
# PTY layer configuration
default_rows = 24
default_cols = 80
enable_terminal_emulation = false
buffer_size = 4096
allocation_timeout_ms = 5000
io_timeout_ms = 1000
cleanup_timeout_ms = 2000

[performance]
# Performance validation settings
max_startup_time_ms = 5000
max_input_response_ms = 100
max_view_switch_ms = 200
memory_limit_mb = 100
min_keys_per_second = 100
min_views_per_second = 100

[validation]
# Validation settings
strict_mode = false
check_performance = true
validate_state_consistency = true
check_integration_points = true
pty_layer_validation = true
keyboard_input_validation = true
component_integration_validation = true

[isolation]
# Test isolation settings
enable_isolation = true
isolated_temp_dir = true
clean_env_vars = true
separate_workspace = true
process_isolation = true

[paths]
# Test paths (relative to project root)
test_data_dir = "test/data"
test_fixtures_dir = "test/fixtures"
test_logs_dir = "test/logs"
test_results_dir = "test/results"
test_workspace_dir = "test/workspace"
test_isolation_dir = "test/isolation"
test_env_dir = "test_env"
EOF
        log_success "Created test configuration: $test_config"
    else
        log_info "Test configuration already exists: $test_config"
    fi

    # Create enhanced PTY configuration
    local pty_config="$TEST_CONFIG_DIR/pty_config.env"
    cat > "$pty_config" << EOF
# PTY Test Environment Configuration
# Generated by TUI test environment setup

# PTY Size Configuration
export PTY_SIZE_COLS=80
export PTY_SIZE_ROWS=24

# PTY Timeout Configuration
export PTY_TIMEOUT=30s
export PTY_ALLOC_TIMEOUT=10s
export PTY_IO_TIMEOUT=5s
export PTY_CLEANUP_TIMEOUT=5s

# PTY Test Configuration
export PTY_TEST_MODE=true
export PTY_DEBUG_MODE=false
export PTY_LOG_LEVEL=info

# Brummer Binary Configuration
export BRUMMER_BINARY=$PROJECT_ROOT/brum
export BRUMMER_NO_TUI=true
export BRUMMER_MCP_ENABLED=false

# Test Environment Configuration
export TEST_ENV=true
export TEST_ISOLATION=true
export TEST_CLEANUP_ON_EXIT=true

# Performance Test Configuration
export PERFORMANCE_TEST_ENABLED=true
export PERFORMANCE_ITERATIONS=1000
export PERFORMANCE_TIMEOUT=30s

# Memory Test Configuration
export MEMORY_TEST_ENABLED=true
export MEMORY_LIMIT_MB=100
export MEMORY_CHECK_INTERVAL=5s

# Log Configuration
export LOG_LEVEL=debug
export LOG_FILE=$PROJECT_ROOT/test/logs/pty_test.log
export LOG_ROTATION=true
EOF

    log_success "Created PTY configuration: $pty_config"

    # Create CI configuration if requested
    if [[ "$SETUP_CI" == "true" ]]; then
        local ci_config="$TEST_CONFIG_DIR/ci_env.toml"

        cat > "$ci_config" << 'EOF'
# CI/CD Test Environment Configuration
[environment]
test_mode = true
verbose_logging = true
pty_required = true
timeout_seconds = 120

[testing]
enable_debug = true
capture_logs = true
keep_test_data = true
parallel_tests = true

[pty]
default_rows = 24
default_cols = 80
enable_terminal_emulation = false
buffer_size = 8192
allocation_timeout_ms = 10000
io_timeout_ms = 2000
cleanup_timeout_ms = 5000

[performance]
max_startup_time_ms = 10000
max_input_response_ms = 200
max_view_switch_ms = 500
memory_limit_mb = 200
min_keys_per_second = 50
min_views_per_second = 50

[validation]
strict_mode = true
check_performance = true
validate_state_consistency = true
check_integration_points = true
pty_layer_validation = true
keyboard_input_validation = true
component_integration_validation = true

[isolation]
enable_isolation = true
isolated_temp_dir = true
clean_env_vars = true
separate_workspace = true
process_isolation = true
EOF
        log_success "Created CI configuration: $ci_config"
    fi
}

# Generate test fixtures
create_test_fixtures() {
    if [[ "$CREATE_FIXTURES" != "true" ]]; then
        log_info "Skipping test fixture creation"
        return 0
    fi

    log "Creating test fixtures..."

    # Create sample package.json for testing
    local sample_package_json="$TEST_FIXTURES_DIR/sample_package.json"
    cat > "$sample_package_json" << 'EOF'
{
  "name": "test-project",
  "version": "1.0.0",
  "description": "Sample project for TUI testing",
  "scripts": {
    "dev": "echo 'Development server started'",
    "build": "echo 'Build completed'",
    "test": "echo 'Tests passed'",
    "start": "echo 'Application started'"
  },
  "dependencies": {
    "express": "^4.18.0",
    "lodash": "^4.17.21"
  },
  "devDependencies": {
    "jest": "^29.0.0",
    "webpack": "^5.0.0"
  }
}
EOF

    # Create test data files
    local test_data_file="$TEST_DATA_DIR/test_scenarios.json"
    cat > "$test_data_file" << 'EOF'
{
  "test_scenarios": [
    {
      "name": "basic_startup",
      "description": "Basic application startup test",
      "expected_duration_ms": 3000,
      "expected_views": ["processes", "logs", "errors"],
      "key_sequence": [
        {"key": "ctrl+c", "delay": 1000}
      ]
    },
    {
      "name": "view_navigation",
      "description": "View switching navigation test",
      "expected_duration_ms": 5000,
      "expected_views": ["processes", "logs", "errors", "ai-coders", "file-browser"],
      "key_sequence": [
        {"key": "1", "delay": 500},
        {"key": "2", "delay": 500},
        {"key": "3", "delay": 500},
        {"key": "4", "delay": 500},
        {"key": "5", "delay": 500},
        {"key": "ctrl+c", "delay": 1000}
      ]
    },
    {
      "name": "command_palette",
      "description": "Command palette functionality test",
      "expected_duration_ms": 4000,
      "expected_views": ["processes"],
      "key_sequence": [
        {"key": "/", "delay": 500},
        {"key": "quit", "delay": 500},
        {"key": "enter", "delay": 500},
        {"key": "ctrl+c", "delay": 1000}
      ]
    }
  ]
}
EOF

    # Create workspace test files
    local workspace_dir="$PROJECT_ROOT/test/workspace/sample-project"
    execute_or_dry_run "Creating workspace sample project" mkdir -p "$workspace_dir"

    if [[ "$DRY_RUN" != "true" ]]; then
        # Copy sample package.json to workspace
        cp "$sample_package_json" "$workspace_dir/package.json"

        # Create additional workspace files
        cat > "$workspace_dir/README.md" << 'EOF'
# Sample Test Project

This is a sample project used for TUI testing.

## Usage

```bash
npm run dev    # Start development server
npm run build  # Build the project
npm test       # Run tests
```
EOF

        cat > "$workspace_dir/.gitignore" << 'EOF'
node_modules/
dist/
*.log
.env
.DS_Store
EOF

        log_success "Created workspace sample project: $workspace_dir"
    fi

    # Create expected state templates
    local expected_states_dir="$TEST_FIXTURES_DIR/expected_states"
    execute_or_dry_run "Creating expected states directory" mkdir -p "$expected_states_dir"

    if [[ "$DRY_RUN" != "true" ]]; then
        cat > "$expected_states_dir/startup_state.json" << 'EOF'
{
  "expected_state": {
    "current_view": "processes",
    "processes_count": {"min": 0, "max": 10},
    "logs_count": {"min": 0, "max": 100},
    "errors_count": 0,
    "ui_elements": {
      "header": true,
      "footer": true,
      "process_list": true,
      "status_bar": true
    }
  },
  "validation_rules": {
    "no_errors": true,
    "responsive_ui": true,
    "proper_layout": true
  }
}
EOF

        cat > "$expected_states_dir/keyboard_input_state.json" << 'EOF'
{
  "expected_state": {
    "input_processed": true,
    "view_changed": false,
    "no_hanging_processes": true,
    "cursor_position": {"valid": true}
  },
  "validation_rules": {
    "input_responsive": true,
    "no_crashes": true,
    "state_consistent": true
  }
}
EOF

        log_success "Created expected state templates"
    fi
}

# Setup test isolation
setup_test_isolation() {
    log "Setting up test isolation..."

    # Create isolation directory
    local isolation_dir="$PROJECT_ROOT/test/isolation"
    execute_or_dry_run "Creating isolation directory: $isolation_dir" mkdir -p "$isolation_dir"

    # Create enhanced test isolation scripts
    local isolation_script="$isolation_dir/isolated_test_runner.sh"
    cat > "$isolation_script" << 'EOF'
#!/bin/bash

# Enhanced Test Isolation Script
# Provides comprehensive isolation for PTY-based TUI tests

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Load configuration
if [[ -f "$PROJECT_ROOT/test/config/pty_config.env" ]]; then
    source "$PROJECT_ROOT/test/config/pty_config.env"
fi

# Isolation configuration
TEST_ID="${TEST_ID:-$(date +%s)_$$}"
ISOLATION_DIR="$PROJECT_ROOT/test/tmp/isolated_$TEST_ID"
ISOLATION_LOG="$PROJECT_ROOT/test/logs/isolation_$TEST_ID.log"
ISOLATION_PID_FILE="$PROJECT_ROOT/test/isolation/pids/test_$TEST_ID.pid"
ISOLATION_SOCKET_DIR="$PROJECT_ROOT/test/isolation/sockets/$TEST_ID"

# Create isolation environment
mkdir -p "$ISOLATION_DIR"
mkdir -p "$(dirname "$ISOLATION_LOG")"
mkdir -p "$(dirname "$ISOLATION_PID_FILE")"
mkdir -p "$ISOLATION_SOCKET_DIR"

# Copy necessary files and configurations
cp -r "$PROJECT_ROOT/test/fixtures" "$ISOLATION_DIR/" 2>/dev/null || true
cp -r "$PROJECT_ROOT/test/data" "$ISOLATION_DIR/" 2>/dev/null || true
cp -r "$PROJECT_ROOT/test/config" "$ISOLATION_DIR/" 2>/dev/null || true

# Set isolated environment variables
export TEST_ISOLATION_ID="$TEST_ID"
export TEST_ISOLATION_DIR="$ISOLATION_DIR"
export TMPDIR="$ISOLATION_DIR"
export HOME="$ISOLATION_DIR"
export XDG_CONFIG_HOME="$ISOLATION_DIR/.config"
export XDG_CACHE_HOME="$ISOLATION_DIR/.cache"
export XDG_DATA_HOME="$ISOLATION_DIR/.local/share"

# Brummer-specific isolation
export BRUMMER_TEST_ISOLATION=true
export BRUMMER_TEST_DIR="$ISOLATION_DIR"
export BRUMMER_CONFIG_FILE="$ISOLATION_DIR/config/test_env.toml"
export BRUMMER_SOCKET_DIR="$ISOLATION_SOCKET_DIR"

# PTY isolation
export PTY_ISOLATION=true
export PTY_WORKSPACE_DIR="$ISOLATION_DIR/pty_workspace"

# Process isolation
export PROCESS_ISOLATION=true
export PROCESS_NAMESPACE="brummer_test_$TEST_ID"

# Cleanup function
cleanup_isolation() {
    echo "Cleaning up test isolation: $TEST_ID"

    # Kill any processes started in this isolation
    if command -v pkill > /dev/null 2>&1; then
        pkill -f "BRUMMER_TEST_ISOLATION_ID=$TEST_ID" || true
        pkill -f "TEST_ISOLATION_ID=$TEST_ID" || true
    fi

    # Clean up isolation directory
    if [[ -d "$ISOLATION_DIR" ]]; then
        rm -rf "$ISOLATION_DIR"
    fi

    # Clean up PID file
    if [[ -f "$ISOLATION_PID_FILE" ]]; then
        rm -f "$ISOLATION_PID_FILE"
    fi

    # Clean up socket directory
    if [[ -d "$ISOLATION_SOCKET_DIR" ]]; then
        rm -rf "$ISOLATION_SOCKET_DIR"
    fi
}

# Signal handlers
trap cleanup_isolation EXIT INT TERM

# Store PID
echo $$ > "$ISOLATION_PID_FILE"

# Log isolation setup
echo "Test isolation established: $TEST_ID"
echo "Isolation directory: $ISOLATION_DIR"
echo "Log file: $ISOLATION_LOG"
echo "Socket directory: $ISOLATION_SOCKET_DIR"

# Change to isolation directory
cd "$ISOLATION_DIR"

# Execute the test command with isolation
if [[ $# -gt 0 ]]; then
    echo "Executing isolated test: $*" | tee -a "$ISOLATION_LOG"
    exec "$@" >> "$ISOLATION_LOG" 2>&1
else
    echo "Usage: $0 <test_command> [args...]"
    exit 1
fi
EOF

    chmod +x "$isolation_script"

    # Create comprehensive test cleanup script
    local cleanup_script="$isolation_dir/cleanup_tests.sh"
    cat > "$cleanup_script" << 'EOF'
#!/bin/bash

# Comprehensive Test Cleanup Script
# Cleans up test artifacts, processes, and temporary files

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Cleanup functions
cleanup_processes() {
    echo "Cleaning up test processes..."

    # Kill any brummer processes started by tests
    if command -v pkill > /dev/null 2>&1; then
        pkill -f "brummer.*--test" || true
        pkill -f "brummer.*test" || true
        pkill -f "BRUMMER_TEST" || true
    fi

    # Clean up any remaining PTY processes
    pkill -f "pty.*test" || true

    # Clean up test isolation processes
    if [[ -d "$PROJECT_ROOT/test/isolation/pids" ]]; then
        for pid_file in "$PROJECT_ROOT/test/isolation/pids"/*.pid; do
            if [[ -f "$pid_file" ]]; then
                local pid=$(cat "$pid_file" 2>/dev/null || echo "")
                if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
                    kill "$pid" 2>/dev/null || true
                fi
                rm -f "$pid_file"
            fi
        done
    fi

    # Clean up any remaining go test processes
    pkill -f "go test.*tty,integration" || true
}

cleanup_temp_files() {
    echo "Cleaning up temporary files..."

    # Clean up isolation temp directories
    if [[ -d "$PROJECT_ROOT/test/tmp" ]]; then
        find "$PROJECT_ROOT/test/tmp" -type d -name "isolated_*" -mtime +1 -exec rm -rf {} + 2>/dev/null || true
        find "$PROJECT_ROOT/test/tmp" -type f -mtime +1 -delete 2>/dev/null || true
        find "$PROJECT_ROOT/test/tmp" -type d -empty -delete 2>/dev/null || true
    fi

    # Clean up log files older than 7 days
    if [[ -d "$PROJECT_ROOT/test/logs" ]]; then
        find "$PROJECT_ROOT/test/logs" -name "*.log" -mtime +7 -delete 2>/dev/null || true
        find "$PROJECT_ROOT/test/logs" -name "isolation_*.log" -mtime +1 -delete 2>/dev/null || true
    fi

    # Clean up test results older than 30 days
    if [[ -d "$PROJECT_ROOT/test/results" ]]; then
        find "$PROJECT_ROOT/test/results" -name "*.json" -mtime +30 -delete 2>/dev/null || true
        find "$PROJECT_ROOT/test/results" -name "*.xml" -mtime +30 -delete 2>/dev/null || true
        find "$PROJECT_ROOT/test/results" -name "*.html" -mtime +30 -delete 2>/dev/null || true
    fi

    # Clean up workspace artifacts
    if [[ -d "$PROJECT_ROOT/test/workspace" ]]; then
        find "$PROJECT_ROOT/test/workspace" -name "*.tmp" -mtime +1 -delete 2>/dev/null || true
        find "$PROJECT_ROOT/test/workspace" -name "*.log" -mtime +7 -delete 2>/dev/null || true
    fi
}

cleanup_socket_files() {
    echo "Cleaning up socket files..."

    # Remove test socket files
    if [[ -d "$PROJECT_ROOT/test/isolation/sockets" ]]; then
        find "$PROJECT_ROOT/test/isolation/sockets" -name "*.sock" -delete 2>/dev/null || true
        find "$PROJECT_ROOT/test/isolation/sockets" -type d -empty -delete 2>/dev/null || true
    fi

    # Remove any remaining socket files in /tmp
    find /tmp -name "brummer_test_*.sock" -mtime +1 -delete 2>/dev/null || true
}

cleanup_environment() {
    echo "Cleaning up environment variables..."

    # Unset test environment variables
    unset TEST_ISOLATION_ID 2>/dev/null || true
    unset TEST_ISOLATION_DIR 2>/dev/null || true
    unset BRUMMER_TEST_ISOLATION 2>/dev/null || true
    unset PTY_ISOLATION 2>/dev/null || true
    unset PROCESS_ISOLATION 2>/dev/null || true
}

cleanup_coverage() {
    echo "Cleaning up coverage files..."

    # Clean up coverage files older than 7 days
    find "$PROJECT_ROOT" -name "*.out" -path "*/test/*" -mtime +7 -delete 2>/dev/null || true
    find "$PROJECT_ROOT" -name "coverage.html" -path "*/test/*" -mtime +7 -delete 2>/dev/null || true
    find "$PROJECT_ROOT" -name "coverage.out" -mtime +7 -delete 2>/dev/null || true
}

# Main cleanup function
main() {
    echo "Starting comprehensive test cleanup..."

    cleanup_processes
    cleanup_temp_files
    cleanup_socket_files
    cleanup_environment
    cleanup_coverage

    echo "Comprehensive test cleanup completed"
}

# Show usage
show_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "OPTIONS:"
    echo "  --processes-only    Clean up processes only"
    echo "  --files-only        Clean up files only"
    echo "  --sockets-only      Clean up sockets only"
    echo "  --env-only          Clean up environment only"
    echo "  --coverage-only     Clean up coverage files only"
    echo "  --dry-run           Show what would be cleaned"
    echo "  -h, --help          Show this help message"
}

# Parse arguments
PROCESSES_ONLY=false
FILES_ONLY=false
SOCKETS_ONLY=false
ENV_ONLY=false
COVERAGE_ONLY=false
DRY_RUN=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --processes-only)
            PROCESSES_ONLY=true
            shift
            ;;
        --files-only)
            FILES_ONLY=true
            shift
            ;;
        --sockets-only)
            SOCKETS_ONLY=true
            shift
            ;;
        --env-only)
            ENV_ONLY=true
            shift
            ;;
        --coverage-only)
            COVERAGE_ONLY=true
            shift
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        -h|--help)
            show_usage
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Run specific cleanup or full cleanup
if [[ "$PROCESSES_ONLY" == "true" ]]; then
    cleanup_processes
elif [[ "$FILES_ONLY" == "true" ]]; then
    cleanup_temp_files
elif [[ "$SOCKETS_ONLY" == "true" ]]; then
    cleanup_socket_files
elif [[ "$ENV_ONLY" == "true" ]]; then
    cleanup_environment
elif [[ "$COVERAGE_ONLY" == "true" ]]; then
    cleanup_coverage
else
    main
fi
EOF

    chmod +x "$cleanup_script"

    # Create PTY-specific isolation script
    local pty_isolation_script="$isolation_dir/pty_isolation.sh"
    cat > "$pty_isolation_script" << 'EOF'
#!/bin/bash

# PTY Test Isolation Script
# Provides PTY-specific isolation for terminal-based tests

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# PTY isolation configuration
PTY_TEST_ID="${PTY_TEST_ID:-pty_$(date +%s)_$$}"
PTY_ISOLATION_DIR="$PROJECT_ROOT/test/tmp/pty_isolated_$PTY_TEST_ID"
PTY_LOG="$PROJECT_ROOT/test/logs/pty_isolation_$PTY_TEST_ID.log"

# Create PTY isolation environment
mkdir -p "$PTY_ISOLATION_DIR"
mkdir -p "$(dirname "$PTY_LOG")"

# PTY-specific environment variables
export PTY_TEST_ID="$PTY_TEST_ID"
export PTY_ISOLATION_DIR="$PTY_ISOLATION_DIR"
export PTY_LOG_FILE="$PTY_LOG"
export PTY_SOCKET_DIR="$PTY_ISOLATION_DIR/sockets"

# Create PTY workspace
mkdir -p "$PTY_ISOLATION_DIR/pty_workspace"
mkdir -p "$PTY_SOCKET_DIR"

# PTY isolation cleanup
cleanup_pty_isolation() {
    echo "Cleaning up PTY isolation: $PTY_TEST_ID"

    # Kill PTY-specific processes
    if command -v pkill > /dev/null 2>&1; then
        pkill -f "PTY_TEST_ID=$PTY_TEST_ID" || true
    fi

    # Clean up PTY isolation directory
    if [[ -d "$PTY_ISOLATION_DIR" ]]; then
        rm -rf "$PTY_ISOLATION_DIR"
    fi
}

trap cleanup_pty_isolation EXIT INT TERM

echo "PTY isolation established: $PTY_TEST_ID"
echo "PTY isolation directory: $PTY_ISOLATION_DIR"

# Execute PTY test command
if [[ $# -gt 0 ]]; then
    echo "Executing PTY isolated test: $*" | tee -a "$PTY_LOG"
    cd "$PTY_ISOLATION_DIR"
    exec "$@" >> "$PTY_LOG" 2>&1
else
    echo "Usage: $0 <pty_test_command> [args...]"
    exit 1
fi
EOF

    chmod +x "$pty_isolation_script"

    log_success "Created comprehensive test isolation scripts"
}

# Validate PTY availability
validate_pty_availability() {
    log "Validating PTY availability..."

    # Test PTY allocation with Go
    local pty_test_result=0
    if timeout 10s go test -tags=tty,integration ./test/tui/pty -v -run TestPTYLayer_AllocationAndBasicFunctionality > /dev/null 2>&1; then
        log_success "PTY layer is available and functional"
        pty_test_result=0
    else
        log_warning "PTY layer test failed"
        pty_test_result=1
    fi

    # Check system PTY support
    if [[ -r /dev/ptmx && -w /dev/ptmx ]]; then
        log_success "System PTY support detected (/dev/ptmx accessible)"
    else
        log_warning "System PTY support may be limited"
    fi

    # Check for required OS features
    case "$(uname -s)" in
        Linux)
            log_info "Linux detected - PTY support should be available"
            ;;
        Darwin)
            log_info "macOS detected - PTY support should be available"
            ;;
        CYGWIN*|MINGW*|MSYS*)
            log_warning "Windows detected - PTY support may be limited"
            ;;
        *)
            log_info "Unknown OS - PTY support status unknown"
            ;;
    esac

    return $pty_test_result
}

# Create helper scripts
create_helper_scripts() {
    log "Creating helper scripts..."

    # Create quick test runner
    local quick_test_script="$PROJECT_ROOT/test/quick_test.sh"
    cat > "$quick_test_script" << 'EOF'
#!/bin/bash

# Quick Test Runner
# Runs a quick subset of TUI tests for development

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Change to project directory
cd "$PROJECT_ROOT"

echo "Running quick TUI tests..."

# Quick contract tests
go test -tags=tty,integration ./test/tui/harness -v -run TestTestHarness_Contract -timeout 30s
go test -tags=tty,integration ./test/tui/harness -v -run TestInputSimulator_Contract -timeout 30s

# Quick PTY test
go test -tags=tty,integration ./test/tui/pty -v -run TestPTYLayer_AllocationAndBasicFunctionality -timeout 30s

# Quick integration test
BRUMMER_BINARY="$PROJECT_ROOT/brum" go test -tags=tty,integration ./test/tui/integration -v -run TestKeyboardInputPTY_Simulation -timeout 30s

echo "Quick tests completed"
EOF

    chmod +x "$quick_test_script"

    # Create development test runner
    local dev_test_script="$PROJECT_ROOT/test/dev_test.sh"
    cat > "$dev_test_script" << 'EOF'
#!/bin/bash

# Development Test Runner
# Runs comprehensive TUI tests for development with detailed output

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Change to project directory
cd "$PROJECT_ROOT"

# Build binary first
make build

echo "Running development TUI tests..."

# Set verbose mode
export VERBOSE=true

# Run test execution script
./test/scripts/run_tui_tests.sh -v -p All

# Run validation script
./test/scripts/validate_tui.sh -v All

echo "Development tests completed"
EOF

    chmod +x "$dev_test_script"

    # Create CI test runner
    local ci_test_script="$PROJECT_ROOT/test/ci_test.sh"
    cat > "$ci_test_script" << 'EOF'
#!/bin/bash

# CI Test Runner
# Runs TUI tests optimized for CI/CD environments

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Change to project directory
cd "$PROJECT_ROOT"

echo "Running CI TUI tests..."

# CI environment settings
export VERBOSE=false
export STRICT_MODE=true
export SETUP_CI=true

# Build binary
make build

# Run tests with CI optimizations
./test/scripts/run_tui_tests.sh -s -r All

# Run validation with strict mode
./test/scripts/validate_tui.sh -s All

# Generate reports in CI format
./test/scripts/validate_tui.sh -p Integration > test_results.json 2>&1

echo "CI tests completed"
EOF

    chmod +x "$ci_test_script"

    log_success "Created helper scripts"
}

# Generate setup report
generate_setup_report() {
    local report_file="$TEST_CONFIG_DIR/setup_report_$(date +%Y%m%d_%H%M%S).txt"

    cat > "$report_file" << EOF
TUI Test Environment Setup Report
=================================
Setup completed: $(date)
Project root: $PROJECT_ROOT
Verbose mode: $VERBOSE
Force rebuild: $FORCE_REBUILD
Skip dependencies: $SKIP_DEPS
Create fixtures: $CREATE_FIXTURES
CI setup: $SETUP_CI
Dry run: $DRY_RUN

Directory Structure:
- Test data: $TEST_DATA_DIR
- Test config: $TEST_CONFIG_DIR
- Test fixtures: $TEST_FIXTURES_DIR
- Test results: $PROJECT_ROOT/test/results
- Test logs: $PROJECT_ROOT/test/logs
- Test workspace: $PROJECT_ROOT/test/workspace

Binary Information:
- Path: $PROJECT_ROOT/brum
- Built: $( ([[ -f "$PROJECT_ROOT/brum" ]] && stat -c %y "$PROJECT_ROOT/brum" 2>/dev/null) || echo "Not built" )

Go Environment:
- Version: $(go version 2>/dev/null || echo "Unknown")
- GOPATH: ${GOPATH:-$(go env GOPATH 2>/dev/null || echo "Unknown")}
- GOCACHE: ${GOCACHE:-$(go env GOCACHE 2>/dev/null || echo "Unknown")}

Helper Scripts Created:
- Quick test: $PROJECT_ROOT/test/quick_test.sh
- Dev test: $PROJECT_ROOT/test/dev_test.sh
- CI test: $PROJECT_ROOT/test/ci_test.sh

Next Steps:
1. Run quick tests: $PROJECT_ROOT/test/quick_test.sh
2. Run full tests: $PROJECT_ROOT/test/scripts/run_tui_tests.sh
3. Validate functionality: $PROJECT_ROOT/test/scripts/validate_tui.sh
4. Check pre-commit hook: $PROJECT_ROOT/githooks/pre-commit-tui-tests.sh

Notes:
- Use $PROJECT_ROOT/test/scripts/setup_test_env.sh --help for setup options
- Test data can be found in: $TEST_DATA_DIR
- Configuration files in: $TEST_CONFIG_DIR
- Fixtures and templates in: $TEST_FIXTURES_DIR
EOF

    log_success "Setup report generated: $report_file"

    # Display summary
    if [[ "$DRY_RUN" != "true" ]]; then
        echo ""
        echo "Setup Summary:"
        echo "=============="
        echo "✓ Directory structure created"
        echo "✓ Go environment validated"
        echo "✓ Dependencies installed"
        echo "✓ Brummer binary built"
        echo "✓ Test configuration created"
        echo "✓ Test fixtures generated"
        echo "✓ Test isolation configured"
        echo "✓ Helper scripts created"
        echo ""
        echo "Ready to run TUI tests!"
        echo ""
        echo "Quick start:"
        echo "  $PROJECT_ROOT/test/quick_test.sh"
        echo ""
        echo "Full test suite:"
        echo "  $PROJECT_ROOT/test/scripts/run_tui_tests.sh"
        echo ""
        echo "Validation:"
        echo "  $PROJECT_ROOT/test/scripts/validate_tui.sh"
    fi
}

# Main execution function
main() {
    log "Starting TUI Test Environment Setup"

    # Parse arguments
    parse_args "$@"

    # Change to project directory
    cd "$PROJECT_ROOT"

    # Execute setup steps
    create_directories
    validate_go_environment
    install_dependencies
    build_binary
    create_test_configuration
    create_test_fixtures
    setup_test_isolation
    validate_pty_availability || true  # Don't fail setup if PTY not available
    create_helper_scripts
    generate_setup_report

    log_success "TUI test environment setup completed!"
}

# Script entry point
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi