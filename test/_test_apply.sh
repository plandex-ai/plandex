#! /bin/bash

set -euo pipefail

echo "Opening application in browser..."
plandex-dev browser "http://localhost:8765/error-test.html" || {
  echo "Browser command failed. Exiting."
  exit 1
}

# Check if python3 is available
# if command -v python3 &> /dev/null; then
#     echo "Starting HTTP server with Python 3..."
#     # Start a simple HTTP server on port 8765
#     python3 -m http.server 8765 &
#     SERVER_PID=$!
    
#     # Give the server a moment to start
#     sleep 1
    
#     # Open the browser to the application
#     echo "Opening application in browser..."
#     plandex-dev browser "http://localhost:8765/error-test.html"
#     echo "plandex-dev browser exit code: $?"
    
#     # If browser command fails, kill the server and exit
#     if [ $? -ne 0 ]; then
#         echo "Browser command failed. Killing server..."
#         kill $SERVER_PID
#         exit 1
#     fi
    
#     # Wait for the server process
#     wait $SERVER_PID
    
# # Check if python is available (for Python 2)
# elif command -v python &> /dev/null; then
#     echo "Starting HTTP server with Python 2..."
#     # Start a simple HTTP server on port 8765
#     python -m SimpleHTTPServer 8765 &
#     SERVER_PID=$!
    
#     # Give the server a moment to start
#     sleep 1
    
#     # Open the browser to the application
#     echo "Opening application in browser..."
#     plandex-dev browser "http://localhost:8765"
#     echo "plandex-dev browser exit code: $?"
    
#     # If browser command fails, kill the server and exit
#     if [ $? -ne 0 ]; then
#         echo "Browser command failed. Killing server..."
#         kill $SERVER_PID
#         exit 1
#     fi
    
#     # Wait for the server process
#     wait $SERVER_PID
    
# else
#     echo "Error: Python is not installed. Please install Python to run the HTTP server."
#     exit 1
# fi