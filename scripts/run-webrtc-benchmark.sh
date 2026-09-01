#!/usr/bin/env bash
set -euo pipefail

: "${LIVECLASS_BENCH_TEACHER_ID:?set LIVECLASS_BENCH_TEACHER_ID}"
: "${LIVECLASS_BENCH_VIEWER_ID:?set LIVECLASS_BENCH_VIEWER_ID}"
: "${LIVECLASS_BENCH_LESSON_ID:?set LIVECLASS_BENCH_LESSON_ID}"

mode="${1:-normal}"
common=(
  -teacher-id "$LIVECLASS_BENCH_TEACHER_ID"
  -viewer-id "$LIVECLASS_BENCH_VIEWER_ID"
  -lesson-id "$LIVECLASS_BENCH_LESSON_ID"
  -environment "${LIVECLASS_BENCH_ENVIRONMENT:-local}"
  -cpu-model "${LIVECLASS_BENCH_CPU_MODEL:-unknown}"
  -memory "${LIVECLASS_BENCH_MEMORY:-unknown}"
  -fps 30 -payload-bytes 1000 -warmup 2s -drain 3s
)

case "$mode" in
  normal)
    go run ./cmd/webrtcbench "${common[@]}" -viewers 100 -connect-workers 20 -duration 10s \
      -scenario sfu-v2-udp-mux-100-viewers -output benchmark-results/webrtc-v2-udp-mux-100-viewers.json
    ;;
  loss-before)
    go run ./cmd/webrtcbench "${common[@]}" -viewers 20 -connect-workers 10 -duration 10s \
      -scenario sfu-v1-loss10-no-nack -output benchmark-results/webrtc-v1-loss10-no-nack.json
    ;;
  loss-after)
    go run ./cmd/webrtcbench "${common[@]}" -viewers 20 -connect-workers 10 -duration 10s \
      -scenario sfu-v2-loss10-nack -output benchmark-results/webrtc-v2-loss10-nack.json
    ;;
  disconnect)
    go run ./cmd/webrtcbench "${common[@]}" -viewers 100 -connect-workers 20 -disconnect-viewers 50 \
      -duration 8s -drain 5s -scenario sfu-v2-disconnect-50-of-100 \
      -output benchmark-results/webrtc-v2-disconnect-50-of-100.json
    ;;
  delayed-audio)
    go run ./cmd/webrtcbench "${common[@]}" -viewers 10 -connect-workers 10 -audio-delay 500ms \
      -warmup 1s -duration 5s -scenario sfu-v2-delayed-audio-track \
      -output benchmark-results/webrtc-v2-delayed-audio-track.json
    ;;
  *)
    echo "usage: $0 {normal|loss-before|loss-after|disconnect|delayed-audio}" >&2
    exit 2
    ;;
esac
