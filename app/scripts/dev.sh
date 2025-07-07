#!/usr/bin/env bash

# Detect zsh and trigger it if its the shell
if [ -n "$ZSH_VERSION" ]; then
  # shell is zsh
  echo "Detected zsh"
  zsh -c "source ~/.zshrc && $*"
fi

# Get the directory of the script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Change to the script directory
cd "$SCRIPT_DIR" || exit 1

# Install Python deps
"$SCRIPT_DIR/litellm_deps.sh"

# Update PATH for python venv
export PATH="$SCRIPT_DIR/../litellm-venv/bin:$PATH"

# Detect if reflex is installed and install it if not
if ! [ -x "$(command -v reflex)" ]; then

  # Check if the $GOPATH is empty
  if [ -z "$GOPATH" ]; then
    echo "Error: GOPATH is not set. Please set it to continue..." >&2
    exit 1
  fi

  echo 'Error: reflex is not installed. Installing it now...' >&2
  go install github.com/cespare/reflex@latest
fi

terminate() {
  pkill -f 'plandex-server' # Assuming plandex-server is the name of your process
  kill -TERM "$pid1" 2>/dev/null
  kill -TERM "$pid2" 2>/dev/null
}

trap terminate SIGTERM SIGINT

(cd .. && cd cli && ./dev.sh)

cd ../

export DATABASE_URL=postgres://ds:@localhost/plandex_local?sslmode=disable
export GOENV=development
export LOCAL_MODE=1

reflex -r '^(cli|shared)/.*\.(go|mod|sum)$' -- sh -c 'cd cli && ./dev.sh' &
pid1=$!

reflex -r '^(server|shared)/.*\.(go|mod|sum|py)$' -s -- sh -c 'cd server && go build && ./plandex-server' &
pid2=$!

wait $pid1
wait $pid2
