#!/usr/bin/env bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VENV_DIR="$SCRIPT_DIR/../litellm-venv"
REQUIRED_PYTHON="python3"
REQUIRED_PACKAGES=("litellm==1.61.1" "fastapi==0.115.12" "uvicorn==0.34.1")

if ! command -v "$REQUIRED_PYTHON" &>/dev/null; then
  echo "Python3 not found. Please install it and run this script again."
  exit 1
fi

if [ ! -d "$VENV_DIR" ]; then
  echo "Creating Python virtual environment at $VENV_DIR..."
  "$REQUIRED_PYTHON" -m venv "$VENV_DIR"
fi

source "$VENV_DIR/bin/activate"

is_installed() {
  python -c "import pkg_resources; pkg_resources.require('$1')" &>/dev/null
}

for package in "${REQUIRED_PACKAGES[@]}"; do
  if ! is_installed "$package"; then
    echo "Installing Python package: $package"
    pip install "$package"
  else
    echo "Python package $package already installed"
  fi
done

deactivate
