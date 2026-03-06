#!/bin/bash
# Differential test runner: compares GoLua vs Lua 5.4.8
# Usage: ./difftest.sh test.lua [timeout_seconds]
set -euo pipefail

LUA_FILE="$1"
TIMEOUT="${2:-5}"
GOLUA="/tmp/golua"

# Run Lua 5.4
LUA_OUT=$(timeout "$TIMEOUT" lua5.4 "$LUA_FILE" 2>&1) && LUA_RC=$? || LUA_RC=$?

# Run GoLua
GO_OUT=$(timeout "$TIMEOUT" "$GOLUA" --timeout $((TIMEOUT * 1000)) "$LUA_FILE" 2>&1) && GO_RC=$? || GO_RC=$?

if [ "$LUA_OUT" = "$GO_OUT" ] && [ "$LUA_RC" = "$GO_RC" ]; then
    echo "PASS: $LUA_FILE"
    exit 0
else
    echo "DIFF: $LUA_FILE"
    echo "--- Lua 5.4 (rc=$LUA_RC) ---"
    echo "$LUA_OUT"
    echo "--- GoLua (rc=$GO_RC) ---"
    echo "$GO_OUT"
    exit 1
fi
