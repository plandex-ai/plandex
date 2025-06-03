#!/usr/bin/env bash

# Get the absolute path to the script's directory, regardless of where it's run from
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

# Change to the app directory if we're not already there
cd "$SCRIPT_DIR"

echo "Checking dependencies..."

if ! [ -x "$(command -v git)" ]; then
    echo 'Error: git is not installed.' >&2
    echo 'Please install git before running this setup script.' >&2
    exit 1
fi

if ! [ -x "$(command -v docker)" ]; then
    echo 'Error: docker is not installed.' >&2
    echo 'Please install docker before running this setup script.' >&2
    exit 1
fi

if ! [ -x "$(command -v docker-compose)" ]; then
    docker compose 2>&1 > /dev/null
    if [[ $? -ne 0 ]]; then
        echo 'Error: docker-compose is not installed.' >&2
        echo 'Please install docker-compose before running this setup script.' >&2
        exit 1
    fi
fi

echo "Starting the local Plandex server and database..."

docker compose pull plandex-server
docker compose up
