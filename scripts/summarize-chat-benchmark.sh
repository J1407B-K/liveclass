#!/usr/bin/env bash
set -euo pipefail

before=(benchmark-results/chat-v1-before-hot-room-100c-5qps-trial{1,2,3}.json)
after=(benchmark-results/chat-v2-after-kafka-low-latency-100c-5qps-trial{1,2,3}.json)

for file in "${before[@]}" "${after[@]}"; do
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
      kafka_publish_mean_ms: stage("10005"; "chat_publish_latency_seconds")
    }
  ' "$@"
}

before_summary="$(summarize "${before[@]}")"
after_summary="$(summarize "${after[@]}")"

jq -n --argjson before "${before_summary}" --argjson after "${after_summary}" '
  def reduction($old; $new): ($old - $new) / $old * 100;
  {
    before: $before,
    after: $after,
    improvement_percent: {
      p50_latency_reduction: reduction($before.p50_ms; $after.p50_ms),
      p95_latency_reduction: reduction($before.p95_ms; $after.p95_ms),
      p99_latency_reduction: reduction($before.p99_ms; $after.p99_ms),
      kafka_publish_latency_reduction: reduction($before.kafka_publish_mean_ms; $after.kafka_publish_mean_ms),
      throughput_increase: (($after.deliveries_per_second - $before.deliveries_per_second) / $before.deliveries_per_second * 100),
      api_cpu_change: (($after.api_cpu_percent - $before.api_cpu_percent) / $before.api_cpu_percent * 100),
      chat_cpu_change: (($after.chat_cpu_percent - $before.chat_cpu_percent) / $before.chat_cpu_percent * 100)
    }
  }
'
