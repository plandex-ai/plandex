#!/bin/bash
# custom-models-test.sh - Plandex custom models functionality test

set -e  # Exit on error

# Source common utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/test_utils.sh"

# Setup for this test
setup() {
    setup_test_dir "custom-models-test"
    
    # Create a simple test file
    echo "package main" > main.go
}

# Set trap for cleanup on exit
trap cleanup_test_dir EXIT

# Create custom models JSON matching the GitHub issue
create_custom_models_json() {
    cat > custom-models.json << 'EOF'
{
  "$schema": "https://plandex.ai/schemas/models-input.schema.json",
  "models": [
    {
      "modelId": "custom-claude-4",
      "publisher": "test",
      "description": "Claude 4 Sonnet test",
      "defaultMaxConvoTokens": 15000,
      "maxTokens": 200000,
      "maxOutputTokens": 64000,
      "reservedOutputTokens": 16000,
      "preferredOutputFormat": "xml",
      "hasImageSupport": true,
      "providers": [
        {
          "provider": "openrouter",
          "modelName": "anthropic/claude-sonnet-4"
        }
      ]
    }
  ],
  "modelPacks": [
    {
      "name": "test-pack",
      "description": "Test model pack",
      "$schema": "https://plandex.ai/schemas/model-pack-inline.schema.json",
      "planner": {
        "modelId": "custom-claude-4",
        "largeContextFallback": "custom-claude-4"
      },
      "architect": "custom-claude-4",
      "coder": "custom-claude-4",
      "summarizer": "custom-claude-4",
      "builder": "custom-claude-4",
      "wholeFileBuilder": "custom-claude-4",
      "names": "custom-claude-4",
      "commitMessages": "custom-claude-4",
      "autoContinue": "custom-claude-4"
    }
  ]
}
EOF
}

main() {
    log "=== Plandex Custom Models Test Started at $(date) ==="
    
    setup

    echo "OPENROUTER_API_KEY: $OPENROUTER_API_KEY"

    run_plandex_cmd "new -n custom-model-test" "Create test plan"
    run_plandex_cmd "models" "Show current models"
    
    log "\n=== Testing Custom Models with Custom Provider ==="
    
    create_custom_models_json
    run_plandex_cmd "models custom --file custom-models.json --save" "Import custom models"
    run_plandex_cmd "models available --custom" "List custom models"
    run_plandex_cmd "set-model test-pack" "Set custom model pack"
    
    # test without required API key
    PREV_KEY=$OPENROUTER_API_KEY
    unset OPENROUTER_API_KEY
    expect_plandex_failure "tell 'write a hello world program in Go'" "Tell with custom models (should fail due to missing API key)"
    
    # restore API key
    export OPENROUTER_API_KEY=$PREV_KEY
    run_plandex_cmd "tell 'write a hello world program in Go'" "Tell with custom models"

    log "\n=== Custom Models Test Completed at $(date) ==="
}

# Run the tests
main