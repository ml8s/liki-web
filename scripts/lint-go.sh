#!/usr/bin/env bash
# Run the pinned golangci-lint over this module, then go vet.
#
# 版本钉在 deps/golangci-lint.version，与 CI golangci-lint-action 共用：
# PATH 上是别的版本会让本地 lint 与 CI 结论不一致，所以版本不符=报错，
# 而不是静默用错版本。
set -eo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PIN_FILE="$ROOT/deps/golangci-lint.version"
WANT="$(sed -n 's/^version: *//p' "$PIN_FILE" | head -1)"
if [ -z "$WANT" ]; then
    echo "golangci-lint pin missing in $PIN_FILE" >&2
    exit 1
fi

INSTALL_HINT="  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b \"\$(go env GOPATH)/bin\" $WANT"

if ! command -v golangci-lint >/dev/null 2>&1; then
    echo "golangci-lint not found; install the pinned version:" >&2
    echo "$INSTALL_HINT" >&2
    exit 1
fi

HAVE="$(golangci-lint version 2>/dev/null | head -1 | awk '{print $4}')"
if [ "${HAVE#v}" != "${WANT#v}" ]; then
    echo "golangci-lint $HAVE does not match the pinned $WANT" >&2
    echo "install the pinned version:" >&2
    echo "$INSTALL_HINT" >&2
    exit 1
fi

cd "$ROOT"
golangci-lint run --timeout=3m ./...
go vet ./...
