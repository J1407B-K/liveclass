# Agent Canonical Eval

主分支只认下面这一套评测输入和输出：

- Gold Set：`eval/cases.jsonl`，固定 50 条；其中 RAG 36、planning 4、routing 3、tool 3、permission 4。
- 语料：`eval/corpus/LiveClass工程课程讲义.md` 与 `internal/rpc/agent/rag/We重邮小程序介绍.md`。
- 运行清单：`eval/canonical-manifest.json`，固定文件哈希、模型、温度、切片参数和索引名。
- RAG 输出：`eval/runs/canonical-rag-{a,b,c,d}.jsonl` 与 `eval/reports/canonical-rag-ablation.json`。
- Agent 目标输出：`eval/runs/canonical-agent-{v1,v2}.jsonl` 与 `eval/reports/canonical-agent-v1-v2.json`。当前供应商推理额度暂停，失败尝试已归档，主目录不保留伪 canonical 文件。
- 历史实验：`eval/archive/`。它们只用于追溯，不再称为 final/current。

`agenteval` 会拒绝重复 Case、重复 Prediction 或缺失 Prediction，避免拿半截运行生成“最终报告”。运行器记录可观察的 skill、tools、retrieved/cited docs、steps、retry、fallback、token 和 latency，不记录私有思维链。

v2 的 planning 不是由 ReAct 自己调用计划工具推进：Advisor 建议 complexity，Runtime 决定 direct 或 planned；planned 路径由结构化 Planner 生成 DAG，Runtime 持久化并逐个调度，每个 step 使用有界 ReAct，结果落库后回到 Executor。`create_task_plan` 仍作为可观察 Trace 能力名供现有 evaluator 判断，但真正的写入权限属于 Runtime。

不依赖模型额度的状态机回归数据记录在 `benchmark-results/agent-adaptive-planning.json`：19 条路由用例、三步调度和崩溃窗口恢复均通过。它验证编排正确性，不能替代下面的 50 条模型质量评测。

## 固定配置

- Chat model：`doubao-seed-2-0-lite-260215`
- Temperature：`0`
- Embedding：`doubao-embedding-vision-251215`，2048 维
- Reranker：`BAAI/bge-reranker-v2-m3`，CPU、FP16 关闭
- Parent / Child / Overlap：`1600 / 400 / 80`；A 单独使用 overlap `0`
- ChildTopK / ParentTopK：`8 / 3`
- lesson_id：`1`

## RAG A/B/C/D

先启动 Qdrant、Elasticsearch，并在本机启动 reranker：

```bash
BGE_RERANK_FP16=false python3 internal/rpc/agent/rerank/bge_server.py
```

分别把两份语料写入三个隔离索引。A 使用 overlap=0；B 使用 overlap=80；C/D 共享 Parent-Child 索引：

```bash
go run ./internal/rpc/agent/workflow/indexer -path internal/rpc/agent/rag/We重邮小程序介绍.md -lesson 1 -overlap 0 -collection lesson_docs_canonical_a_v4 -index lesson_docs_canonical_a_v4
go run ./internal/rpc/agent/workflow/indexer -path eval/corpus/LiveClass工程课程讲义.md -lesson 1 -overlap 0 -collection lesson_docs_canonical_a_v4 -index lesson_docs_canonical_a_v4
go run ./internal/rpc/agent/workflow/indexer -path internal/rpc/agent/rag/We重邮小程序介绍.md -lesson 1 -overlap 80 -collection lesson_docs_canonical_b_v4 -index lesson_docs_canonical_b_v4
go run ./internal/rpc/agent/workflow/indexer -path eval/corpus/LiveClass工程课程讲义.md -lesson 1 -overlap 80 -collection lesson_docs_canonical_b_v4 -index lesson_docs_canonical_b_v4
go run ./internal/rpc/agent/workflow/indexer -path internal/rpc/agent/rag/We重邮小程序介绍.md -lesson 1 -overlap 80 -collection lesson_docs_canonical_c_v4 -index lesson_docs_canonical_c_v4
go run ./internal/rpc/agent/workflow/indexer -path eval/corpus/LiveClass工程课程讲义.md -lesson 1 -overlap 80 -collection lesson_docs_canonical_c_v4 -index lesson_docs_canonical_c_v4

go run ./cmd/rageval -variant baseline -collection lesson_docs_canonical_a_v4 -index lesson_docs_canonical_a_v4 -output eval/runs/canonical-rag-a.jsonl
go run ./cmd/rageval -variant baseline_overlap -collection lesson_docs_canonical_b_v4 -index lesson_docs_canonical_b_v4 -output eval/runs/canonical-rag-b.jsonl
go run ./cmd/rageval -variant parent_child -collection lesson_docs_canonical_c_v4 -index lesson_docs_canonical_c_v4 -output eval/runs/canonical-rag-c.jsonl
go run ./cmd/rageval -variant parent_child_rerank -rerank-stage parent -collection lesson_docs_canonical_c_v4 -index lesson_docs_canonical_c_v4 -output eval/runs/canonical-rag-d.jsonl
go run ./cmd/agenteval -category rag -predictions A=eval/runs/canonical-rag-a.jsonl,B=eval/runs/canonical-rag-b.jsonl,C=eval/runs/canonical-rag-c.jsonl,D=eval/runs/canonical-rag-d.jsonl -output eval/reports/canonical-rag-ablation.json
```

## Agent v1/v2

启动固定配置的 Agent；`AGENT_EVAL_ALLOW_V1` 只开放评测兼容分支：

```bash
AGENT_EVAL_ALLOW_V1=true \
LIVECLASS_AGENT_DOC_COLLECTION=lesson_docs_canonical_c_v4 \
LIVECLASS_AGENT_DOC_INDEX=lesson_docs_canonical_c_v4 \
LIVECLASS_AGENT_CHAT_TEMPERATURE=0 \
go run ./internal/rpc/agent
```

每一轮使用新的 run-id，避免会话历史污染：

```bash
go run ./cmd/agentreplay -variant v1 -run-id canonical-20260903-final -user <eval-user-id> -output eval/runs/canonical-agent-v1.jsonl
go run ./cmd/agentreplay -variant v2 -run-id canonical-20260903-final -user <eval-user-id> -output eval/runs/canonical-agent-v2.jsonl
go run ./cmd/agenteval -predictions v1=eval/runs/canonical-agent-v1.jsonl,v2=eval/runs/canonical-agent-v2.jsonl -output eval/reports/canonical-agent-v1-v2.json
```

评测回放连续 3 个 RPC error 会提前终止，防止供应商故障时继续消耗完整数据集并生成误导报告。RAG D 若 reranker 不可用会直接失败，不会静默伪装成 C。评测结论必须同时报告质量和延迟；本数据集仍是项目级离线 Gold Set，不代表通用问答能力。
