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
    echo 'Error: docker-compose is not installed.' >&2
    echo 'Please install docker-compose before running this setup script.' >&2
    exit 1
fi

cp _env .env

echo "Setup Plandex, please add custom details to the .env file that has been created ..."

echo "For security and stability please add custom values to the .env file that has been created ..."

echo "Setup complete!"

echo "Starting the application..."

docker compose build
docker compose up