#!/bin/bash
# test-utils.sh - Common utilities for Plandex test scripts

export PLANDEX_ENV='development'

# Colors for output
export RED='\033[0;31m'
export GREEN='\033[0;32m'
export YELLOW='\033[1;33m'
export NC='\033[0m' # No Color

# Default command
export PLANDEX_CMD="${PLANDEX_CMD:-plandex-dev}"

# Logging functions
log() {
    echo -e "$1"
}

success() {
    log "${GREEN}✓ $1${NC}"
}

error() {
    log "${RED}✗ $1${NC}"
    exit 1
}

info() {
    log "${YELLOW}→ $1${NC}"
}

# Run command and check for success
run_cmd() {
    local cmd="$1"
    local description="$2"
    
    info "Running: $cmd"
    
    # Run command and capture output and exit code properly
    set +e  # Temporarily disable exit on error
    output=$(eval "$cmd" 2>&1)
    local exit_code=$?
    set -e  # Re-enable exit on error
    
    # Log the output
    echo "$output"
    
    if [ "$exit_code" -eq 0 ]; then
        success "$description"
    else
        error "$description failed (exit code: $exit_code)"
    fi
}

# Run plandex command
run_plandex_cmd() {
    local cmd="$1"
    local description="$2"
    run_cmd "$PLANDEX_CMD $cmd" "$description"
}

# Run plandex command and check if output contains substring
check_plandex_contains() {
    local cmd="$1"
    local expected="$2"
    local description="$3"
    
    info "Running: $PLANDEX_CMD $cmd"
    
    local output=$($PLANDEX_CMD $cmd 2>&1)
    echo "$output"
    
    if echo "$output" | grep -q "$expected"; then
        success "$description"
    else
        error "$description - expected to find '$expected'"
    fi
}

# Check if command fails (expecting failure)
expect_failure() {
    local cmd="$1"
    local description="$2"
    
    info "Running (expecting failure): $cmd"
    
    # Run the command and capture both output and exit code
    set +e  # Temporarily disable exit on error
    output=$(eval "$cmd" 2>&1)
    local exit_code=$?
    set -e  # Re-enable exit on error
    
    echo "$output"
    
    if [ "$exit_code" -ne 0 ]; then
        success "$description (failed as expected with exit code $exit_code)"
    else
        error "$description should have failed but succeeded (exit code: $exit_code)"
    fi
}

# Expect plandex command to fail
expect_plandex_failure() {
    local cmd="$1"
    local description="$2"
    expect_failure "$PLANDEX_CMD $cmd" "$description"
}

# Check if file exists
check_file() {
    if [ -f "$1" ]; then
        success "File exists: $1"
    else
        error "File missing: $1"
    fi
}

# Setup test environment
setup_test_dir() {
    source ../.env.client-keys

    local test_name="$1"
    TEST_DIR="/tmp/plandex-${test_name}-$$"
    TIMESTAMP=$(date +%Y%m%d_%H%M%S)
    
    info "Setting up test environment in $TEST_DIR"
    mkdir -p "$TEST_DIR"
    cd "$TEST_DIR"
    
    success "Test environment created"
}

# Cleanup function
cleanup_test_dir() {
    info "Cleaning up test environment"
    cd /
    rm -rf "$TEST_DIR"
    success "Cleanup complete"
}