# Agent v1 vs Agent v2 Report

Run at 2026-09-02 (Asia/Shanghai) using the same seven cases, user, local
dependencies, configured model, and automatic trace collector. The v1
compatibility baseline disables Transcript recovery/compaction, episodic
retrieval, and Parent expansion, while retaining the same security and Trace
instrumentation so process metrics remain observable. It is enabled only with
`AGENT_EVAL_ALLOW_V1=true`.

| Metric | v1 baseline | v2 |
|---|---:|---:|
| Task success | 71.4% | 85.7% |
| Deterministic answer correctness | 90.5% | 85.7% |
| Skill accuracy | 100% | 100% |
| Tool accuracy | 85.7% | 100% |
| Constraint violation rate | 0% | 0% |
| Mean latency | 11.275 s | 13.563 s |
| Mean tokens | 1926.1 | 1927.7 |
| Mean LLM calls | 1.0 | 1.0 |
| Mean tool calls | 0.0 | 0.286 |
| Mean observable steps | 3.57 | 4.14 |
| Fallback rate | 0% | 0% |
| Tool error rate | 0% | 0% |

v2 improved task success by 14.3 percentage points and tool accuracy by 14.3
points because the complex planning case created and updated a persisted
TaskPlan. It did not improve every metric: deterministic answer correctness was
4.8 points lower due one RAG answer omitting required course-service details,
and mean latency was 2.288 s (20.3%) higher. Token use was effectively neutral.
The dataset is only seven cases, so these are regression results rather than a
general quality or performance claim.

The main response calls used 11,788 input / 1,695 output tokens in v1 and
12,056 input / 1,438 output tokens in v2. At the configured
Doubao-Seed-2.0-lite 0--32k list price (CNY 0.6/M input and CNY 3.6/M output),
the measured main-call cost changes from CNY 0.0131748 to CNY 0.0124104
(-5.8%). Total tokens differ by only +0.08%, so the defensible product-level
conclusion is that Agent v2 improves the experience while cost remains
effectively neutral. This estimate excludes uninstrumented Advisor, profile,
and fact-extraction model calls and therefore is not a full-agent saving claim.

Raw predictions are `eval/runs/agent-v1.jsonl` and
`eval/runs/agent-v2.jsonl`; the exact machine-readable report is
`eval/reports/agent-v1-v2.json`.
