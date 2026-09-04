#!/usr/bin/env bash
set -euo pipefail

before=(benchmark-results/chat-v1-before-hot-room-100c-5qps-trial{1,2,3}.json)
after=(benchmark-results/chat-v2-after-kafka-low-latency-100c-5qps-trial{1,2,3}.json)
idempotent=(benchmark-results/chat-v3-after-correctness-100c-5qps-trial{1,2,3}.json)
durable=(benchmark-results/chat-v4-outbox-100c-5qps-trial{1,2,3}.json)
client_dedup=(benchmark-results/chat-v5-client-dedup-100c-5qps-trial{1,2,3}.json)

for file in "${before[@]}" "${after[@]}" "${idempotent[@]}" "${durable[@]}" "${client_dedup[@]}"; do
  test -s "${file}" || { echo "missing result: ${file}" >&2; exit 1; }
done

summarize() {
  jq -s '
    def mean: add / length;
    def endpoint($port): "http://127.0.0.1:" + $port + "/metrics";
    def stage($port; $name):
      map(
        (.prometheus_samples[endpoint($port)].last[$name + "_sum"] -
         .prometheus_samples[endpoint($port)].first[$name + "_sum"]) /
        (.prometheus_samples[endpoint($port)].last[$name + "_count"] -
         .prometheus_samples[endpoint($port)].first[$name + "_count"]) * 1000
      ) | mean;
    {
      trials: length,
      p50_ms: (map(.fanout_latency_ms.p50) | mean),
      p95_ms: (map(.fanout_latency_ms.p95) | mean),
      p99_ms: (map(.fanout_latency_ms.p99) | mean),
      deliveries_per_second: (map(.fanout_deliveries_per_second) | mean),
      error_rate_percent: (map(.delivery_error_rate_percent) | mean),
      api_cpu_percent: (map(.prometheus_samples[endpoint("10001")].average_process_cpu_percent) | mean),
      chat_cpu_percent: (map(.prometheus_samples[endpoint("10005")].average_process_cpu_percent) | mean),
      api_peak_heap_mib: (map(.prometheus_samples[endpoint("10001")].peak.go_memstats_heap_alloc_bytes) | max / 1048576),
      chat_peak_heap_mib: (map(.prometheus_samples[endpoint("10005")].peak.go_memstats_heap_alloc_bytes) | max / 1048576),
      api_peak_goroutines: (map(.prometheus_samples[endpoint("10001")].peak.go_goroutines) | max),
      chat_peak_goroutines: (map(.prometheus_samples[endpoint("10005")].peak.go_goroutines) | max),
      redis_rate_limit_mean_ms: stage("10001"; "chat_redis_rate_limit_latency_seconds"),
      rpc_mean_ms: stage("10001"; "chat_rpc_latency_seconds"),
      fanout_mean_ms: stage("10001"; "chat_fanout_latency_seconds"),
      mongo_mean_ms: stage("10005"; "chat_mongo_latency_seconds"),
      kafka_publish_mean_ms: stage("10005"; "chat_publish_latency_seconds"),
      outbox_published: (map((.prometheus_samples[endpoint("10005")].last.chat_outbox_published_total // 0) - (.prometheus_samples[endpoint("10005")].first.chat_outbox_published_total // 0)) | add),
      outbox_retries: (map((.prometheus_samples[endpoint("10005")].last.chat_outbox_retry_total // 0) - (.prometheus_samples[endpoint("10005")].first.chat_outbox_retry_total // 0)) | add),
      duplicate_deliveries_suppressed: (map((.prometheus_samples[endpoint("10001")].last.chat_duplicate_deliveries_suppressed_total // 0) - (.prometheus_samples[endpoint("10001")].first.chat_duplicate_deliveries_suppressed_total // 0)) | add)
    }
  ' "$@"
}

before_summary="$(summarize "${before[@]}")"
after_summary="$(summarize "${after[@]}")"
idempotent_summary="$(summarize "${idempotent[@]}")"
durable_summary="$(summarize "${durable[@]}")"
client_dedup_summary="$(summarize "${client_dedup[@]}")"

jq -n --argjson before "${before_summary}" --argjson after "${after_summary}" --argjson idempotent "${idempotent_summary}" --argjson durable "${durable_summary}" --argjson client_dedup "${client_dedup_summary}" '
  def reduction($old; $new): ($old - $new) / $old * 100;
  {
    before: $before,
    after: $after,
    idempotent: $idempotent,
    durable_outbox: $durable,
    client_side_dedup: $client_dedup,
    improvement_percent: {
      p50_latency_reduction: reduction($before.p50_ms; $after.p50_ms),
      p95_latency_reduction: reduction($before.p95_ms; $after.p95_ms),
      p99_latency_reduction: reduction($before.p99_ms; $after.p99_ms),
      kafka_publish_latency_reduction: reduction($before.kafka_publish_mean_ms; $after.kafka_publish_mean_ms),
      throughput_increase: (($after.deliveries_per_second - $before.deliveries_per_second) / $before.deliveries_per_second * 100),
      api_cpu_change: (($after.api_cpu_percent - $before.api_cpu_percent) / $before.api_cpu_percent * 100),
      chat_cpu_change: (($after.chat_cpu_percent - $before.chat_cpu_percent) / $before.chat_cpu_percent * 100)
    },
    correctness_change_percent: {
      p50_latency: (($client_dedup.p50_ms - $after.p50_ms) / $after.p50_ms * 100),
      p95_latency: (($client_dedup.p95_ms - $after.p95_ms) / $after.p95_ms * 100),
      p99_latency: (($client_dedup.p99_ms - $after.p99_ms) / $after.p99_ms * 100),
      throughput: (($client_dedup.deliveries_per_second - $after.deliveries_per_second) / $after.deliveries_per_second * 100)
    },
    end_to_end_percent: {
      p50_latency_reduction: reduction($before.p50_ms; $client_dedup.p50_ms),
      p95_latency_reduction: reduction($before.p95_ms; $client_dedup.p95_ms),
      p99_latency_reduction: reduction($before.p99_ms; $client_dedup.p99_ms),
      throughput_increase: (($client_dedup.deliveries_per_second - $before.deliveries_per_second) / $before.deliveries_per_second * 100)
    }
  }
'
