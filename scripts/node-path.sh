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
    # 非 nvm 环境（CI 用 actions/setup-node，node 在 toolcache+PATH）：退回 PATH 的 node。
    if command -v node >/dev/null 2>&1; then
      echo "node-path.sh: v$NODE_VERSION not in nvm, using PATH node ($(command -v node))" >&2
      NODE_BIN="$(cd "$(dirname "$(command -v node)")" && pwd)"
    else
      echo "node-path.sh: v$NODE_VERSION not found in $NVM_DIR/versions/ and no node in PATH" >&2
      echo "Run: nvm install" >&2
      exit 1
    fi
  fi
  export NODE_BIN
  export PATH="$NODE_BIN:$PATH"
else
  echo "node-path.sh: .nvmrc not found" >&2
  exit 1
fi
