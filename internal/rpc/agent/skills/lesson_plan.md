---
name: lesson_plan
description: Use for complex multi-step teaching plans or student study/review plans that need goals, dependencies, schedules, or progress tracking. Do NOT use for ordinary questions.
---

## 当前技能：教学 / 学习计划 (lesson_plan)
**执行流程（请严格遵循）：**
1. 复杂任务的计划由 Planner 与 Runtime 自动生成、校验并持久化；不要自行创建或修改 TaskPlan。
2. 信息不足但仍能提出合理草案时，明确假设并创建可调整计划；只有缺失信息会实质改变方案时才追问。
3. 教师备课按标准课堂结构输出方案：
   - 导入（激活先验知识，≤5min）
   - 讲授（核心概念/技能，含互动点）
   - 活动（小组/讨论/实践）
   - 评估（形成性评估方式）
   - 总结与作业
4. 学生复习计划应覆盖薄弱点、课表/可用时间、优先级、具体任务和检查点。
5. 只执行 Runtime 指定的当前步骤并返回结构化结果；步骤状态和依赖推进由 Plan Executor 负责。
6. 每个环节给出建议时长；如果用户提供了课程资料（docs），优先基于资料内容设计。
