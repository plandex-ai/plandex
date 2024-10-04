#!/bin/bash

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

# make sure that we are in the same directory as the script
cd "$(dirname "$0")"

# copy the _env file to .env unless it already exists
if [ -f .env ]; then
    echo ".env file already exists, won't overwrite it with _env"
    echo "Add any custom values to .env"
else
    echo "Copying _env file to .env"
    cp _env .env
    echo ".env has been populated with default values"
    echo "Add any custom values to .env"
fi

echo "Setup complete!"

echo "Starting the application..."

docker compose build
docker compose up
