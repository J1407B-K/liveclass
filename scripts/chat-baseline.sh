#!/usr/bin/env bash
set -euo pipefail

: "${LIVECLASS_CHAT_TOKEN:?set LIVECLASS_CHAT_TOKEN to a valid access token}"
: "${LIVECLASS_LESSON_ID:?set LIVECLASS_LESSON_ID to a lesson containing that user}"

mkdir -p benchmark-results
timestamp="$(date -u +%Y%m%dT%H%M%SZ)"

go run ./cmd/chatbench \
  -lesson-id "${LIVECLASS_LESSON_ID}" \
  -connections "${CHAT_CONNECTIONS:-100}" \
  -connect-workers "${CHAT_CONNECT_WORKERS:-50}" \
  -qps "${CHAT_QPS:-5}" \
  -duration "${CHAT_DURATION:-60s}" \
  -warmup "${CHAT_WARMUP:-10s}" \
  -drain "${CHAT_DRAIN:-5s}" \
  -metrics "${CHAT_METRICS:-http://127.0.0.1:10001/metrics,http://127.0.0.1:10005/metrics}" \
  -environment "${CHAT_ENVIRONMENT:-local}" \
  -cpu-model "${CHAT_CPU_MODEL:-}" \
  -memory "${CHAT_MEMORY:-}" \
  -redis "${CHAT_REDIS_DEPLOYMENT:-}" \
  -mongo "${CHAT_MONGO_DEPLOYMENT:-}" \
  -kafka "${CHAT_KAFKA_DEPLOYMENT:-}" \
  -output "benchmark-results/chat-baseline-${timestamp}.json"
