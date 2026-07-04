#!/usr/bin/env bash
# Build the morgan WebAssembly adapter.
set -euo pipefail
cd "$(dirname "$0")"
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" ./wasm_exec.js
GOOS=js GOARCH=wasm go build -ldflags="-s -w" -o morgan.wasm .
echo "built morgan.wasm ($(du -h morgan.wasm | cut -f1))"
