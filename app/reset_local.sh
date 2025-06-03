#!/usr/bin/env bash

# Get the absolute path to the script's directory, regardless of where it's run from
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

# Change to the app directory if we're not already there
cd "$SCRIPT_DIR"

echo "Clearing local mode..."

./clear_local.sh

echo "Starting local mode..."

./start_local.sh