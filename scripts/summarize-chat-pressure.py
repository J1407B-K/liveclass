#!/usr/bin/env python3
"""Summarize completed pressure runs without treating configured QPS as achieved QPS."""
import json
from pathlib import Path
import sys

root = Path(__file__).resolve().parents[1]
directory = root / (sys.argv[1] if len(sys.argv)>1 else 'benchmark-results/chat-pressure-20260928')
state = json.loads((directory/'state.json').read_text())
rows = []
for path in sorted(directory.glob('*.json')):
    data = json.loads(path.read_text())
    if not isinstance(data,dict) or 'workload' not in data or 'smoke' in path.name:
        continue
    work = data['workload']
    mongo = path.with_name(path.stem+'-mongo.json')
    accepted = published = pending = None
    if mongo.exists():
        collections = json.loads(mongo.read_text())
        message_collections = [item for item in collections if item.get('collection') == 'messages']
        accepted = sum(item['count'] for item in message_collections)
        states = [item for coll in message_collections for item in coll['outbox']]
        if any(item['_id'] is not None for item in states):
            published = sum(item['count'] for item in states if item['_id']=='published')
            pending = sum(item['count'] for item in states if item['_id'] in ['pending','publishing'])
    latency = data['fanout_latency_ms']
    metrics = data.get('prometheus_samples',{}).get('http://127.0.0.1:10005/metrics',{})
    last_metrics = metrics.get('last', {})
    rows.append({
        'run':path.stem,'commit':data['source_commit'],'connections':work['connections'],
        'opened':data['connections_opened'],'rooms':len(work['lesson_ids']), 'duration':work['duration'],
        'target_qps':work['messages_per_second'],'actual_send_qps':data['actual_send_rate'],
        'sent':data['messages_sent'],'accepted_mongo':accepted,'published_mongo':published,
        'pending_after_drain':pending,'peak_pending':metrics.get('peak',{}).get('chat_outbox_pending'),
        'delivered':data['fanout_deliveries_received'],'expected':data['fanout_deliveries_expected'],
        'delivery_deficit_pct':data['delivery_error_rate_percent'],
        'load_deliveries_per_second':data['fanout_deliveries_per_second'],
        'p50_ms':latency['p50'],'p99_ms':latency['p99'],
        'rejected_acks':data['rejected_acknowledgements'],'protocol_errors':data['protocol_errors'],
        'read_errors':data['read_errors'],'duplicate_deliveries':data['duplicate_deliveries'],
        'send_errors':data['send_errors'],'connection_errors':data['connection_errors'],
        'ordering_defers':last_metrics.get('chat_outbox_ordering_deferred_total', 0),
        'ordering_checks':last_metrics.get('chat_outbox_ordering_checks_total', 0),
    })
(directory/'summary.json').write_text(json.dumps(rows,indent=2)+'\n')
text = ['# Chat 多版本压力测试（2026-09-28）','',
'## 口径','',
'- 本轮使用独立 Docker 项目，实际 WebSocket → JWT → 课堂成员权限 RPC/MySQL → Redis 限流 → Chat/Mongo → Kafka → API 广播全链路。',
'- 1,000 个隔离测试用户各持有一个有效 JWT，轮流发送；未关闭限流。用户/课堂关联直接在测试数据库初始化，不包含注册登录耗时。',
'- 每条内容 128 字节；热点课堂 1,000 个接收连接，多课堂为 10 个课堂各 100 个连接。',
'- Kafka topic 固定 3 个分区，API/Chat 各一个进程。本轮最新 affinity 代码使用 4 个 Outbox worker；可用 `--outbox-workers` 做诊断对照。',
'- 每轮预热等待 5 秒；负载停止后最多等待 30 秒接收尾部消息。30 秒阶梯用于寻找退化点，180 秒档位用于持续验证。',
'- actual_send_qps 是实际成功写入客户端 WebSocket 的速率，不能替代服务端接受量。accepted_mongo 是本轮独立数据库中的消息数。',
'- 旧内存队列版本可能在 Mongo 写入后发布入队失败，因此 Mongo 条数不等于 RPC 成功确认数；新版另有 acknowledgements 和 rejected_acknowledgements。',
'- 投递缺口包含未接受、仍积压和连接断开造成的缺口，不直接等于永久丢失。published 表示 Kafka 确认及 Outbox 标记完成；旧内存队列版本无此状态，显示 —。',
'- `pending_after_drain` 来自接收窗口结束后的 Mongo 快照，查询期间后台仍可继续发布。Prometheus 的 pending gauge 在 worker 持续 drain 时可能陈旧，不能据其峰值判断真实最大积压。',
'- 广播吞吐只使用负载窗口内收到的唯一消息数；最终投递量包含 drain。重复投递单独统计。P99 仅针对收到的消息，缺失消息不在延迟样本中。分位数为 1ms 桶的上界。',
'- 所有服务与压测端共享同一台 Apple M4 / 16GB。Docker 分配约 7.75GiB、10 CPU；Go 1.27.0，历史库提示 JSON 回退。没有独立负载机，也未模拟真实网络。',
'- MongoDB 7.0、Kafka 3.9.1、Redis Stack 7.4、MySQL 8.0；镜像摘要、源码提交、工具哈希见 state.json。Jaeger 未启动，链路仍有追踪开销及导出失败，不与旧环境数字直接合并。','',
'## 版本来源','',
'使用真实 Git 快照及各自 go.mod，没有把当前代码修改后冒充历史版本。历史实验 v3/v4 没有独立可对应的源码提交，因此本轮没有伪造两行结果。','',
'| 本轮名称 | 提交 | 实现阶段 |','|---|---|---|']
descriptions = {'v1':'早期同步广播与默认 Kafka 批等待','v2':'低延迟 Kafka + 有界内存队列/Room','v5':'Mongo Outbox + 连接级去重，未加入课堂排序闸门','ordered':'当前版本，增加 Mongo 前序消息排序闸门'}
for label,value in state['versions'].items():
    text.append(f"| {label} | `{value['commit'][:7]}` | {descriptions[label]} |")
text += ['', '## Lesson affinity worker','', '旧模型是多个 worker 从全局 pending outbox 直接 claim；同一 lesson 会被多个 worker 同时拿到，再通过 `HasEarlierUnpublished` 检查并 `DeferForOrdering`。新模型在 claim 阶段按 `lesson_id % worker_count` 做 worker affinity：同一个 lesson 固定进入同一个 worker，严格串行；不同 lesson 可以并行。`HasEarlierUnpublished` 仍保留，作为跨实例安全兜底。','', '## 已完成结果','', '| 场景 | 连接成功 | 目标/实际输入 msg/s | 时长 | Mongo 接受 | Kafka published | 排空后 pending | 最终投递/期望 | 缺口 | P99 ms | ordering defer |', '|---|---:|---:|---|---:|---:|---:|---:|---:|---:|---:|']
display = lambda value: '—' if value is None else str(value)
for r in rows:
    text.append(f"| [{r['run']}]({r['run']}.json) | {r['opened']}/{r['connections']} | {r['target_qps']:g}/{r['actual_send_qps']:.1f} | {r['duration']} | {display(r['accepted_mongo'])} | {display(r['published_mongo'])} | {display(r['pending_after_drain'])} | {r['delivered']}/{r['expected']} | {r['delivery_deficit_pct']:.2f}% | {r['p99_ms']:.0f} | {r['ordering_defers']:.0f} |")
text += ['', '本轮 affinity 结果与上一轮全局单 worker结果对比：hot50 仍为 100% 投递，P99 86ms（上一轮 36ms）；ordering defer 从 83,462 降到 0。multi100 仍为 100% 投递，P99 42ms（上一轮 23ms）；multi500 缺口从 21.06% 降到 13.83%，P99 基本持平；multi1000 仍是当前容量上限，缺口约 83.19%。本轮 ordering checks 是实际调用 `HasEarlierUnpublished` 的次数，ordering defer 是实际调用 `DeferForOrdering` 的次数。', '', '每轮的 `*-mongo.json`、`*-metrics.json`、`*-kafka-lag.txt` 保存数据库状态、终态指标与消费组 lag；主 JSON 内的 timeline 保存逐秒积压和资源样本。', '', '## 复现','', '```bash', 'docker compose -f scripts/chat-pressure.compose.yml up -d', 'GOCACHE=/private/tmp/liveclass-gocache go build -o /private/tmp/liveclass-pressure-webrtc ./internal/rpc/webrtc_live', 'python3 scripts/chat-pressure.py prepare --output benchmark-results/chat-pressure-new', 'python3 scripts/chat-pressure.py smoke --output benchmark-results/chat-pressure-new', 'python3 scripts/chat-pressure.py run --output benchmark-results/chat-pressure-new', 'python3 scripts/chat-pressure.py run --output benchmark-results/chat-pressure-new --versions ordered --cases hot50 --duration 180', 'python3 scripts/summarize-chat-pressure.py benchmark-results/chat-pressure-new', '```', '', '脚本只停止它启动的服务进程。Docker 测试容器保留，停止命令：`docker compose -f scripts/chat-pressure.compose.yml stop`。数据库测试记录未删除。']
(directory/'README.md').write_text('\n'.join(text)+'\n')
print(json.dumps(rows,ensure_ascii=False,indent=2))
