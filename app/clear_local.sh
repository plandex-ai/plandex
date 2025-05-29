#!/usr/bin/env bash

# Get the absolute path to the script's directory, regardless of where it's run from
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

# Change to the app directory if we're not already there
cd "$SCRIPT_DIR"

echo "WARNING: This will delete all Plandex server data and reset the database."
echo "This action cannot be undone."
read -p "Are you sure you want to continue? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]
then
    echo "Reset cancelled."
    exit 1
fi

echo "Resetting local mode..."
echo "Stopping containers and removing volumes..."

# Stop containers and remove volumes
docker compose down -v

echo "Database and data directories cleared. Server stopped."