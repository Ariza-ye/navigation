#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUTPUT="${OUTPUT:-$ROOT_DIR/bin/navigation}"
GOCACHE="${GOCACHE:-$ROOT_DIR/.gocache}"

mkdir -p "$(dirname "$OUTPUT")"
mkdir -p "$GOCACHE"

cd "$ROOT_DIR"
GOCACHE="$GOCACHE" CGO_ENABLED="${CGO_ENABLED:-1}" go build -trimpath -ldflags="-s -w" -o "$OUTPUT" .

echo "Built $OUTPUT"
