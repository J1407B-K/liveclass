# 消息接受语义与幂等

课堂聊天把 MongoDB 持久化成功定义为消息已经被系统接受，而不是把 Kafka 实时广播成功定义为接受。一次发送先校验用户和课堂，再用服务端生成的 UUIDv7 作为 message_id 写入 MongoDB；客户端另外提供稳定的 client_message_id。数据库为 sender_id 与 client_message_id 建立唯一的部分索引，只有携带该字段的消息参与约束。这样网络超时后，客户端可以用原来的 client_message_id 重试，服务端原子地返回第一次写入的 message_id，不会再次插入，也不会重复发布 Kafka。

Outbox 子文档直接嵌入 message，初始状态是 pending，因此一次 Mongo 单文档原子 upsert 同时完成消息保存和待发布事件创建。RPC 返回 accepted，delivery_status 是 outbox_pending；命中已有幂等记录时是 duplicate。唤醒 Relay 的内存通知只用于降低延迟，通知丢失后轮询仍能从 Mongo 找回 pending，不会出现消息已落库但发布任务永久丢失。

幂等键的作用域不是整个系统，而是发送者加客户端消息号。相同发送者复用同一 client_message_id 却改变 lesson_id 或 content，会被判定为冲突；不同发送者可以使用相同字符串。未携带 client_message_id 的旧客户端仍按传统方式插入，每次请求都会形成新消息，这只用于兼容迁移期。历史查询以服务端 message_id 和 created_at 为准，不能拿 Kafka 是否广播成功判断数据是否存在。

# Kafka 实时广播边界

Chat API 使用 Kafka 的目标是缓冲突发聊天、解耦 Chat 持久化与 WebSocket fanout，并跨 API 实例广播。每个 API 实例都要收到所有课堂消息，所以实例之间不能共享一个 consumer group；共享 group 会把一条消息只分配给一个实例，导致其他实例连接的学生收不到。系统使用 chat-api 和稳定实例标识组成 group id，使每个实例独立订阅并保存自己的 offset。

默认模式是 durable_replay。Reader 对已有稳定 group 从已提交 offset 恢复；新实例首次创建 group 时从 LastOffset 开始，连接到新实例的客户端再通过 Mongo cursor 补历史。Pod 短暂重启后可以消费停机窗口的 backlog，不必等待仍在线的客户端主动发现缺口。Kafka backlog 可能带来恢复洪峰，因此 WebSocket 仍使用有界队列和慢消费者隔离。

Outbox 与 durable consumer 都是 at-least-once：Kafka 写成功但 Mongo published 状态回写失败时会在租约过期后重发，consumer 广播后提交 offset 失败也会重放。API 先把记录广播到 Room，再提交 Kafka offset；每个 WebSocket Client 用有界 message-ID 集合抑制重复投递。这样不需要在 Redis 先写去重标记，也就没有“Redis 已标记、广播前崩溃”造成漏发的窗口。部署配置统一提供 broker、topic、group_prefix 和 fanout_mode。

# WebSocket 并发与慢消费者

每个 WebSocket Client 只有一个 readPump 和一个 writePump。readPump 负责读取客户端消息、刷新读截止时间并调用发送处理逻辑；writePump 是该连接唯一的写者，负责写数据帧和定时 Ping。Room 广播不能直接调用连接的 WriteMessage，而是把序列化消息投递到每个 Client 自己的有界 send 队列。这样某个学生网络慢，只会让自己的队列积压，不会卡住遍历房间的广播协程。

队列必须有容量上限，因为无限队列只是把背压伪装成内存增长。队列满时执行 slow consumer policy：增加 slow_consumer_total 和 dropped_messages_total，并注销、关闭该连接。客户端随后通过重连和历史消息恢复。这个策略牺牲单个慢连接的瞬时连续性，换取房间内健康连接的延迟稳定和服务端内存上界。

连接生命周期还包括 read limit、Pong handler、应用层 idle timer、write deadline 和 graceful cleanup。Hertz netpoll WebSocket 不支持直接依赖 SetReadDeadline，因此服务器发送 Ping，收到数据或 Pong 后重置 idle timer；超时后主动取消连接。注销逻辑必须幂等，确保连接只从 Room 删除一次。一个课堂最后一个连接离开后，RoomManager 回收空 Room，避免课堂数量增长造成常驻对象泄漏。

# 限流与发布背压

聊天入口使用 Redis Lua 脚本原子完成固定窗口计数与过期时间设置，避免 GET、INCR、EXPIRE 分步执行时出现竞态。限流键包含用户和课堂维度，目标是限制单个发送者，而不是让一个活跃用户耗尽整个课堂额度。Lua 返回是否允许以及剩余额度；Redis 调用必须有超时，并记录 chat_redis_rate_limit_latency_seconds。

RPC 写入 Mongo 后不会为每条消息启动一个不受控 goroutine，也不把可靠性押在内存队列上。固定数量 Outbox Relay worker 用 FindOneAndUpdate 原子领取 pending 或租约过期的 publishing 记录，写 Kafka 时使用 write_timeout；成功才更新 published，失败记录 attempts、last_error 和 next_attempt_at。Mongo 的 pending 数量就是持久背压，进程重启后仍然存在。

指数退避表示第 n 次等待在基础时间上按倍数增长，并加入随机抖动，减少多个实例同时恢复时的惊群。单轮 Kafka 写只做有限次快速重试，跨轮失败由 Mongo next_attempt_at 调度并设置最大退避上限。outbox pending、claimed、published、retry、客户端重复抑制和 publish latency 都要暴露指标。这里的重点不是保证 Kafka 永不失败，而是让请求快速 Accepted，同时让积压和恢复过程可观察、可重启。

# 历史查询与顺序保证

聊天系统承诺的是持久化历史的确定顺序，不宣称跨课堂全局有序。UUIDv7 带时间有序特征，created_at 记录服务端接受时间；历史分页使用稳定游标，排序同时考虑时间和 message_id，避免相同时间戳导致翻页重复或遗漏。多个 Relay worker 或 Chat 实例并发时，Kafka key 不能单独证明严格实时全序，因此客户端最终展示也使用 created_at、message_id 排序。

WebSocket 重连携带 last_message_id。服务端先注册连接并把此时到达的 Kafka 消息放进该 Client 的有界 pendingLive，再从 Mongo 查询 anchor 之后的消息，按升序写入后冲刷实时缓冲。Client 用有上限的 message-ID 集合消除历史与实时重叠；每批最多一百条、单次最多自动补一千条，超出返回 truncated，避免无限占用内存。

确认帧包含 client_message_id、服务端 message_id 和 delivery_status。outbox_pending 表示消息与发布任务已经原子持久化；duplicate 表示数据库已有对应请求。面试中应把 accepted、published、delivered 三个阶段分开陈述，因为它们分别对应 Mongo 事实、Kafka broker 确认和终端网络送达。

# 依赖治理与熔断

Agent 把超时、重试、退避和熔断集中为每个依赖的 policy，覆盖主模型、Embedding、Reranker、Qdrant、Elasticsearch、Postgres、Redis 和 Kafka。读请求通常可以有限重试，非幂等写请求默认不自动重试。统一入口记录 dependency、operation、attempt、latency、fallback 等观测信息，避免每个调用点自行 sleep 和重复包装。

熔断器有 Closed、Open、HalfOpen 三种状态。Closed 正常放行并在滚动窗口统计失败；达到最小请求数且失败比例超过阈值后转为 Open，立即拒绝后续调用。Open 不是永久不通，open_duration 到期后转为 HalfOpen，只允许少量探测请求。探测成功达到要求就恢复 Closed，探测失败则重新 Open。它保护的是本服务线程和下游依赖，不是声称微服务业务可以凭空绕过数据库。

降级必须符合业务语义。Reranker 不可用时可以退回未精排的检索结果，Elasticsearch 不可用时可以退回向量检索；但 Postgres 关键写入失败不能伪造成功。长期记忆写入使用 transactional outbox：事实表和 outbox event 在同一数据库事务提交，后台 worker 再投递 Kafka，失败记录 retry_count 与 last_error。这样解决数据库成功但事件丢失的双写问题。

# 自适应 Plan-and-Execute

Agent 的 Advisor 只建议技能和复杂度，Runtime 才是是否建立计划的最终裁决者。简单解释、单次查询或改写直接运行普通 ReAct，避免额外 Planner 调用增加延迟和 Token；同时包含规划意图与多步骤、依赖、阶段或时间安排的复杂任务，才进入 Plan-and-Execute。

Planner 是一次只输出结构化 JSON 的模型调用，不调用工具。Runtime 校验计划有二到六个步骤、step key 合法、依赖存在并且 DAG 无环，再把 TaskPlan 持久化。Executor 只选择依赖已经完成的 ready step，把单个步骤交给有最大迭代次数的 ReAct；StepResult 先写入 Transcript，再把步骤标记 done，然后控制权必须返回 Executor。失败最多重新规划一次，并保留已完成步骤。

StepResult 先于 done 状态持久化是为了处理崩溃窗口。如果进程写完结果却尚未更新步骤状态，重启后 Executor 能识别 running step 已有结果，只补状态而不重复调用工具。模型不能直接创建或修改 TaskPlan，状态跳转、依赖约束、恢复和重规划都属于 Runtime 不变量。

# Parent Child RAG 与评测

文档索引先按 Markdown 标题切成 parent，再把每个 parent 切为更小的 child。检索阶段对 child 建立向量索引和 BM25 索引，使用相同 lesson_id 过滤后融合结果；命中的 child 再按 parent_id 扩展并去重，把完整上下文交给模型。child 小有利于精确召回，parent 大有利于回答时保留定义、条件和例子。

child overlap 用来缓解答案跨切片边界的问题，但 overlap 也会增加索引体积和重复召回。Reranker 可在 child、parent 或 two_stage 阶段运行。parent rerank 输入更完整但文本更长、延迟更高；child rerank 更快但可能只看到局部证据；two_stage 先筛 child 再排 parent，成本最高。任何“更好”的结论都必须由固定数据集的 Hit@1、Recall@3、MRR 与 latency 支持。

规范评测固定 cases.jsonl、模型名、检索配置、索引语料和运行标识。RAG 至少使用三十条问题，整体 Agent 用例至少五十条，并同时覆盖 routing、tool、permission 和 planning。旧实验移动到 archive，主目录只保留 canonical 输出。小于十条的数据只能称为 regression smoke test，不能拿偶然的一两条变化宣称显著提升。

# WebRTC SFU 反馈恢复

SFU 不像 MCU 那样解码后混流，它接收老师发布的 RTP 包，再把编码包转发给每个学生的本地 Track。一个 remote track 只代表一条媒体轨道，音频和视频分别绑定；同一发布者的音视频通过课堂和发布者关系组成 BroadcastBundle。学生加入时创建自己的 PeerConnection 和 local tracks，remote 读到的 RTP 分别写入所有匹配的 local track。

Pion 的 NACK responder interceptor 会缓存近期发送的 RTP 包。学生检测到序列号缺口后通过 RTCP TransportLayerNack 请求重传，发送端从缓存中找到对应包再次发送。PLI/FIR 不要求重发某个旧包，而是请求老师编码器尽快产生新关键帧，适合解码状态已经无法恢复的情况。服务器把学生侧 PLI 转发给老师的 Publisher PeerConnection。

千人课堂不能让一千个学生同时触发一千个上行 PLI，因此 BroadcastBundle 对 PLI 做最小间隔聚合：间隔内重复请求计入 suppressed，只向老师转发一次。TWCC 提供传输层拥塞反馈，为后续码率自适应提供基础；UDP mux 让多个 PeerConnection 复用监听 socket，减少端口和文件描述符压力。压测需要分别验证无丢包、注入丢包、延迟音轨和批量断连，不能只证明信令链路连通。
