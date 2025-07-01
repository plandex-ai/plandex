#!/bin/bash

# Plandex Smoke Test Script
# Tests core functionality in a linear flow mimicking real usage
# Assumes: Already signed in to Plandex Cloud (staging account)

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test directory setup
TEST_DIR="/tmp/plandex-smoke-test-$$"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
LOG_FILE="${TEST_DIR}/smoke-test-${TIMESTAMP}.log"

PLANDEX_CMD="plandex-dev"

# Minimal prompts to keep costs down
PROMPT_CREATE_FUNCTION="add a simple hello world function in main.go"
PROMPT_ADD_TEST="add a test for the hello function"
PROMPT_CHAT_QUESTION="what does the hello function do?"
PROMPT_ADD_FEATURE="add a goodbye function that returns: goodbye world"

# Helper functions
log() {
    if [ -f "$LOG_FILE" ]; then
        echo -e "$1" | tee -a "$LOG_FILE"
    else
        echo -e "$1"
    fi
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
    
    # Create a temporary file for capturing exit status
    local tmpfile=$(mktemp)
    
    # Run command with output visible and logged
    ( eval "$cmd" 2>&1; echo $? > "$tmpfile" ) | tee -a "$LOG_FILE"
    
    # Get the exit status
    local exit_code=$(cat "$tmpfile")
    rm -f "$tmpfile"
    
    if [ "$exit_code" -eq 0 ]; then
        success "$description"
    else
        error "$description failed (exit code: $exit_code)"
    fi
}

# Run command and capture output
run_cmd_output() {
    local cmd="$1"
    
    # Create a temporary file for capturing exit status
    local tmpfile=$(mktemp)
    
    if [ -f "$LOG_FILE" ]; then
        ( eval "$cmd" 2>&1; echo $? > "$tmpfile" ) | tee -a "$LOG_FILE"
    else
        ( eval "$cmd" 2>&1; echo $? > "$tmpfile" )
    fi
    
    # Get the exit status
    local exit_code=$(cat "$tmpfile")
    rm -f "$tmpfile"
    
    # Return the exit code so the caller can handle it
    return $exit_code
}

run_plandex_cmd() {
    local cmd="$1"
    local description="$2"
    run_cmd "$PLANDEX_CMD $cmd" "$description"
}

run_plandex_cmd_output() {
    local cmd="$1"
    if ! run_cmd_output "$PLANDEX_CMD $cmd"; then
        error "Command failed: $PLANDEX_CMD $cmd"
    fi
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
setup() {
    info "Setting up test environment in $TEST_DIR"
    mkdir -p "$TEST_DIR"
    cd "$TEST_DIR"
    
    # Now create the log file after directory exists
    LOG_FILE="${TEST_DIR}/smoke-test-${TIMESTAMP}.log"
    touch "$LOG_FILE"
    
    # Create a simple Go project structure
    mkdir -p cmd
    echo "package main" > main.go
    echo "func main() {}" >> main.go
    
    # Create a test file to load as context
    cat > README.md << EOF
# Test Project
This is a test project for Plandex smoke testing.
EOF
    
    success "Test environment created"
}

# Cleanup function
cleanup() {
    info "Cleaning up test environment"
    cd /
    rm -rf "$TEST_DIR"
    success "Cleanup complete"
}

# Set trap for cleanup on exit
trap cleanup EXIT

# Main test flow
main() {
    log "=== Plandex Smoke Test Started at $(date) ==="
    
    setup
    
    # 1. PLAN MANAGEMENT
    log "\n=== Testing Plan Management ==="
    
    # Create new plan with name
    run_plandex_cmd "new -n smoke-test-plan" "Create named plan"
    
    # Check current plan
    run_plandex_cmd_output "current"
    
    # List plans
    run_plandex_cmd_output "plans"
    
    # 2. CONTEXT MANAGEMENT
    log "\n=== Testing Context Management ==="
    
    # Load single file
    run_plandex_cmd "load main.go" "Load single file"
    
    # Load with note
    run_plandex_cmd "load -n 'keep code simple and well-commented'" "Load note"
    
    # Load directory tree
    run_plandex_cmd "load . --tree" "Load directory tree"
    
    # List context
    run_plandex_cmd_output "ls"
    
    # Show specific context
    run_plandex_cmd_output "show main.go"
    
    # 3. BASIC TASK EXECUTION
    log "\n=== Testing Task Execution ==="

    # Skip changes menu so we don't have to interact with the menu
    run_plandex_cmd "set-config skip-changes-menu true" "Set skip-changes-menu to true"
    
    # Tell command with simple task
    run_plandex_cmd "tell '$PROMPT_CREATE_FUNCTION'" "Execute tell command"
    
    # Check diff
    run_plandex_cmd_output "diff --git"
    
    # Apply changes
    run_plandex_cmd "apply --auto-exec --debug 2 --skip-commit" "Apply changes"
    
    # Verify file was updated
    check_file "main.go"
    
    # 4. CHAT FUNCTIONALITY
    log "\n=== Testing Chat ==="
    
    # Chat without making changes
    run_plandex_cmd_output "chat '$PROMPT_CHAT_QUESTION'"
    
    # 5. CONTINUE AND BUILD
    log "\n=== Testing Continue and Build ==="
    
    # Tell another task
    run_plandex_cmd "tell '$PROMPT_ADD_TEST' --no-build" "Tell without building"
    
    # Build pending changes
    run_plandex_cmd "build" "Build pending changes"
    
    # Review and apply
    run_plandex_cmd_output "diff --git"
    run_plandex_cmd "apply --auto-exec --debug 2 --skip-commit" "Apply test changes"
    
    # 6. BRANCHES
    log "\n=== Testing Branches ==="
    
    # Create and switch to new branch
    run_plandex_cmd "checkout feature-branch -y" "Create new branch"
    
    # Make changes on branch
    run_plandex_cmd "tell '$PROMPT_ADD_FEATURE'" "Add feature on branch"
    run_plandex_cmd "apply --auto-exec --debug 2 --skip-commit" "Apply on branch"
    
    # List branches
    run_plandex_cmd_output "branches"
    
    # Switch back to main
    run_plandex_cmd "checkout main" "Switch to main branch"
    
    # 7. VERSION CONTROL
    log "\n=== Testing Version Control ==="
    
    # View log
    run_plandex_cmd_output "log"
    
    # View conversation
    run_plandex_cmd_output "convo"
    
    # Get current state for rewind test
    REWIND_STEPS=2
    info "Will rewind $REWIND_STEPS steps"
    
    # Rewind
    run_plandex_cmd "rewind $REWIND_STEPS" "Rewind $REWIND_STEPS steps"
    
    # 8. CONFIGURATION
    log "\n=== Testing Configuration ==="
    
    # View current config
    run_plandex_cmd_output "config"
    
    # Change a setting
    run_plandex_cmd "set-config auto-continue false" "Set auto-continue to false"

    # Change it back
    run_plandex_cmd "set-config auto-continue true" "Set auto-continue to true"
    
    # View models
    run_plandex_cmd_output "models"
    
    # List model packs
    run_plandex_cmd_output "model-packs"
    
    # 9. CONTEXT UPDATES
    log "\n=== Testing Context Updates ==="
    
    # Modify a file outside of Plandex
    echo "// Modified outside Plandex" >> main.go
    
    # Update context
    run_plandex_cmd "update" "Update outdated context"
    
    # Remove context
    run_plandex_cmd "rm main.go" "Remove file from context"
    
    # Clear all context
    run_plandex_cmd "clear" "Clear all context"
    
    # 10. REJECT FUNCTIONALITY
    log "\n=== Testing Reject ==="
    
    # Load context again and make changes
    run_plandex_cmd "load . -r" "Reload context"
    run_plandex_cmd "tell 'add a function that has an intentional syntax error'" "Create changes to reject"
    
    # Reject all pending changes
    run_plandex_cmd "reject --all" "Reject all pending changes"
    
    # 11. ARCHIVE FUNCTIONALITY
    log "\n=== Testing Archive ==="
    
    # Archive the plan
    run_plandex_cmd "archive smoke-test-plan" "Archive plan"
    
    # List archived plans
    run_plandex_cmd_output "plans --archived"
    
    # Unarchive
    run_plandex_cmd "unarchive smoke-test-plan" "Unarchive plan"
    
    # 12. MULTIPLE PLANS
    log "\n=== Testing Multiple Plans ==="
    
    # Create another plan with model pack
    run_plandex_cmd "new -n second-plan --cheap" "Create plan with cheap model pack"
    
    # Switch between plans
    run_plandex_cmd "cd smoke-test-plan" "Switch to first plan"
    run_plandex_cmd_output "current"
    
    log "\n=== Plandex Smoke Test Completed Successfully at $(date) ==="
}

# Run the tests
main