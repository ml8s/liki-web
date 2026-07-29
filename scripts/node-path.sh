#!/usr/bin/env bash
# Get the project's Node.js binary path.
# Source this file: source scripts/node-path.sh
# Then use $NODE_BIN/node, $NODE_BIN/npx, etc.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
NVM_DIR="${NVM_DIR:-$HOME/.nvm}"

if [ -f "$PROJECT_DIR/.nvmrc" ]; then
  NODE_VERSION=$(cat "$PROJECT_DIR/.nvmrc")
  NODE_BIN="$NVM_DIR/versions/node/v$NODE_VERSION/bin"
  if [ ! -d "$NODE_BIN" ]; then
    echo "node-path.sh: v$NODE_VERSION not found in $NVM_DIR/versions/" >&2
    echo "Run: nvm install" >&2
    exit 1
  fi
  export NODE_BIN
  export PATH="$NODE_BIN:$PATH"
else
  echo "node-path.sh: .nvmrc not found" >&2
  exit 1
fi
