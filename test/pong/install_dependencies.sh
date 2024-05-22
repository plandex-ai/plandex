#!/bin/bash

# Check for Homebrew and install if not found
if ! command -v brew &> /dev/null
then
    echo "Homebrew not found. Installing Homebrew..."
    /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
fi

# Install necessary dependencies
echo "Installing necessary dependencies..."
brew install freeglut

echo "Dependencies installed successfully."
