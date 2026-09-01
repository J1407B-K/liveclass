#!/usr/bin/env bash
set -euo pipefail

go run ./cmd/resiliencebench \
  -requests 200 \
  -concurrency 10 \
  -dependency-delay 100ms \
  -timeout 20ms \
  -attempts 2 \
  -backoff 5ms \
  -open-duration 100ms \
  -trials 3 \
  -cpu-model "Apple M4 (10 cores)" \
  -memory "16GB" \
  -output benchmark-results/resilience-fault-injection.json
