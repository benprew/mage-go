#!/bin/bash
# Build the mage WASM module and copy wasm_exec.js support file.
set -euo pipefail

cd "$(dirname "$0")/.."

echo "Building WASM..."
GOOS=js GOARCH=wasm go build -o web/mage.wasm ./cmd/wasm/

echo "Copying wasm_exec.js..."
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" web/

echo "Done. Serve web/ with any HTTP server, e.g.:"
echo "  cd web && python3 -m http.server 8080"
