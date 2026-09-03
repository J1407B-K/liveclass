# Agent v2 Offline Eval

`cases.jsonl` 是唯一 Gold Dataset。RAG A/B/C/D 与 Agent v1/v2 必须使用同一份文件，禁止手工修改预测结果。

预测文件每行字段见 `internal/rpc/agent/eval.Prediction`。Agent 在线运行器应从可观察 Trace 生成 skill、tools、steps、retry、fallback、token 和 latency；不得记录或评估私有 Chain-of-Thought。

RAG 消融：

```bash
go run ./internal/rpc/agent/workflow/indexer -path internal/rpc/agent/rag/We重邮小程序介绍.md -lesson 1 -overlap 0 -collection lesson_docs_eval_a -index lesson_docs_eval_a
go run ./internal/rpc/agent/workflow/indexer -path internal/rpc/agent/rag/We重邮小程序介绍.md -lesson 1 -overlap 80 -collection lesson_docs_eval_b -index lesson_docs_eval_b
go run ./internal/rpc/agent/workflow/indexer -path internal/rpc/agent/rag/We重邮小程序介绍.md -lesson 1 -overlap 80 -collection lesson_docs_eval_c -index lesson_docs_eval_c
go run ./cmd/rageval -variant baseline -collection lesson_docs_eval_a -index lesson_docs_eval_a -output eval/runs/rag-a.jsonl
go run ./cmd/rageval -variant baseline_overlap -collection lesson_docs_eval_b -index lesson_docs_eval_b -output eval/runs/rag-b.jsonl
go run ./cmd/rageval -variant parent_child -collection lesson_docs_eval_c -index lesson_docs_eval_c -output eval/runs/rag-c.jsonl
go run ./cmd/rageval -variant parent_child_rerank -collection lesson_docs_eval_c -index lesson_docs_eval_c -output eval/runs/rag-d.jsonl
go run ./cmd/agenteval -category rag -predictions A=eval/runs/rag-a.jsonl,B=eval/runs/rag-b.jsonl,C=eval/runs/rag-c.jsonl,D=eval/runs/rag-d.jsonl -output eval/reports/rag-ablation.json
```

A 与 B 使用隔离的 overlap=0/80 索引，C/D 共享 Parent-Child 索引。真实依赖未启动时不生成结果报告；D 在 reranker 不可用时直接失败，不会静默伪装成 C。

Rerank stage 对比可通过 `-rerank-stage child|parent|two_stage` 复现；三组必须使用相同 `parent_top_k`。当前实测报告为 `eval/reports/rag-rerank-strategies.json`。

Agent v1/v2：

```bash
AGENT_EVAL_ALLOW_V1=true go run ./internal/rpc/agent
go run ./cmd/agentreplay -variant v1 -run-id <unique-run-id> -user <eval-user-id> -output eval/runs/agent-v1.jsonl
go run ./cmd/agentreplay -variant v2 -run-id <unique-run-id> -user <eval-user-id> -output eval/runs/agent-v2.jsonl
go run ./cmd/agenteval -predictions v1=eval/runs/agent-v1.jsonl,v2=eval/runs/agent-v2.jsonl -output eval/reports/agent-v1-v2.json
```

v1 兼容分支只允许在显式环境变量开启时使用；它关闭 v2 Session/Compaction/Episodic/Parent Expansion，但保留共同的权限与 Trace 外壳以采集可比指标。每轮必须使用新的 `run-id`，否则同一 Case 的历史会污染后续回放。

报告中的 Faithfulness 是严格的 evidence-grounded citation score：引用必须同时出现在检索结果和 gold_docs。更通用的语义正确性可以追加同一套 rubric 的 LLM Judge，但不得替换确定性指标。
