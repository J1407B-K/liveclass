# RAG Ablation Report

Run at 2026-09-01 (Asia/Shanghai) on Apple M4 / 16 GB, with local Qdrant,
Elasticsearch, the configured embedding model, and CPU `bge-reranker-v2-m3`.
All four variants used the same four `category=rag` cases from
`eval/cases.jsonl`. Raw predictions and the machine-readable report are under
`eval/runs/` and `eval/reports/rag-ablation.json`.

| Variant | Pipeline | Hit@1 | Recall@3 | Recall@5 | MRR | Mean retrieval latency |
|---|---|---:|---:|---:|---:|---:|
| A | child retrieval, overlap 0 | 0.50 | 0.75 | 0.75 | 0.661 | 238.5 ms |
| B | child retrieval, overlap 80 | 0.50 | 0.75 | 0.75 | 0.661 | 221.5 ms |
| C | parent expansion and dedup | 0.50 | 0.75 | 0.75 | 0.661 | 252.7 ms |
| D | parent expansion plus parent rerank | 0.75 | 0.75 | 0.75 | 0.750 | 724.9 ms |

On this small document every markdown section fits in one child chunk, so
overlap and parent expansion cannot improve A/B/C retrieval quality. This is a
dataset limitation, not evidence that those techniques never help. Parent-level
reranking improved Hit@1 by 0.25 absolute and MRR by about 0.089, while adding
about 472 ms mean latency versus C. No throughput, CPU, or memory improvement is
claimed from this run.

Reproduce by creating the three isolated indexes and running the commands in
`eval/README.md`. Index IDs are deterministic, so reruns replace the same
points instead of accumulating duplicates.

## Rerank stage comparison

With the same final `parent_top_k=3`, child-, parent-, and two-stage reranking
all produced Hit@1 0.75, Recall@3/5 0.75, and MRR 0.75. Their mean retrieval
latencies were 845.3 ms, 748.0 ms, and 1042.3 ms respectively. The runtime
therefore keeps parent-level reranking: it matched the other strategies on this
gold set and was the least expensive. See `rag-rerank-strategies.json` for the
machine-readable result.
