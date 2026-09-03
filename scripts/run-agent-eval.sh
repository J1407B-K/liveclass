#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 2 ]]; then
  echo "usage: $0 OUTPUT_JSON variant=predictions.jsonl [variant=predictions.jsonl ...]" >&2
  exit 2
fi

output=$1
shift
predictions=$(IFS=,; echo "$*")
GOCACHE="${GOCACHE:-/tmp/liveclass-gocache}" go run ./cmd/agenteval \
  -cases eval/cases.jsonl \
  -predictions "$predictions" \
  -output "$output"
