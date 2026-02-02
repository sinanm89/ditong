#!/bin/bash
# Ditong Benchmark Suite Runner
# Usage: ./run.sh [group] [iterations]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

GROUP="${1:-}"
ITERATIONS="${2:-1}"

echo "=================================="
echo "Ditong Benchmark Suite"
echo "=================================="
echo ""

# Build ditong if needed
if [ ! -f "../go/ditong" ] && [ ! -f "../go/ditong.exe" ]; then
    echo "Building ditong..."
    cd ../go
    go build -o ditong ./cmd
    cd "$SCRIPT_DIR"
fi

# Run benchmarks
if [ -n "$GROUP" ]; then
    echo "Running group: $GROUP"
    go run runner.go --group "$GROUP" --iterations "$ITERATIONS" --force
else
    echo "Running all groups..."
    go run runner.go --iterations "$ITERATIONS" --force
fi

echo ""
echo "Done!"
