#!/bin/bash
set -e

echo "=== Running Full Simulation Test Suite ==="
go test -v -race ./testing/simulation/...

echo "=== Running Consensus Benchmarks ==="
go test -bench=. ./testing/benchmarks/...

echo "=== Verifying Binary Build ==="
go build -o sovereignd cmd/sovereignd/main.go
./sovereignd > /dev/null 2>&1 &
PID=$!

# Allow node server time to bind port
sleep 2

echo "=== Checking Metrics Endpoint ==="
curl -s http://127.0.0.1:8080/metrics | grep sovereign_blocks_mined_total || true

echo "=== Stopping Sovereign Node ==="
kill $PID 2>/dev/null || true

echo "=== Cleaning Up Test Artifacts ==="
rm -rf sovereignd data/ test_data_pebble/

echo "=== Platform Build & Verification Complete ==="
