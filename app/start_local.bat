@echo off

echo Checking dependencies...

where git >nul 2>&1
if %errorlevel% neq 0 (
    echo Error: git is not installed. >&2
    echo Please install git before running this setup script. >&2
    exit /b 1
)

where docker >nul 2>&1
if %errorlevel% neq 0 (
    echo Error: docker is not installed. >&2
    echo Please install docker before running this setup script. >&2
    exit /b 1
)

where docker-compose >nul 2>&1
if %errorlevel% neq 0 (
    echo Error: docker-compose is not installed. >&2
    echo Please install docker-compose before running this setup script. >&2
    exit /b 1
)

REM copy the _env file to .env unless it already exists
if exist .env (
    echo .env file already exists, won't overwrite it with _env
    echo Add any custom values to .env
) else (
    echo Copying _env file to .env
    copy _env .env
    echo .env has been populated with default values
    echo Add any custom values to .env
)

echo Setup complete!

echo Starting the application...

docker compose build
docker compose up