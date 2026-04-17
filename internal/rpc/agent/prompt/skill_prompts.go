package prompt

// SkillType 定义支持的技能类型
type SkillType = string

const (
	SkillStudentQA     SkillType = "student_qa"     // 学生答疑、概念理解
	SkillLessonPlan    SkillType = "lesson_plan"    // 教学设计、备课
	SkillQuizHelp      SkillType = "quiz_help"      // 测验出题、习题讲解
	SkillLessonSummary SkillType = "lesson_summary" // 课堂总结、知识点回顾
	SkillGeneral       SkillType = "general"        // 通用对话
)

// SkillPrompts 是各技能对应的业务级执行 SOP，会被注入到 agent 的上下文中，
// 告诉 React Agent 应该按什么流程来处理当前这类任务。
var SkillPrompts = map[SkillType]string{
	SkillStudentQA: `
## 当前技能：学生答疑 (student_qa)
**执行流程（请严格遵循）：**
1. 先用一句话确认自己理解了学生的问题。
2. 如果是开放题或评估题，**不要直接给出答案**，改为提问引导：
   "你目前已知道哪些信息？你尝试过哪些思路？"
3. 识别学生的认知卡点，用类比、图示描述或具体例子拆解。
4. 分步骤解释核心概念，每一步确认学生是否理解后再推进。
5. 最后给出 1-2 个可操作的巩固建议（练习题、复习要点等）。
`,
	SkillLessonPlan: `
## 当前技能：教学设计 (lesson_plan)
**执行流程（请严格遵循）：**
1. 先确认：教学对象（年级/层次）、课时长度、教学目标。
2. 按标准课堂结构输出方案：
   - 导入（激活先验知识，≤5min）
   - 讲授（核心概念/技能，含互动点）
   - 活动（小组/讨论/实践）
   - 评估（形成性评估方式）
   - 总结与作业
3. 每个环节给出建议时长。
4. 附上差异化建议（针对不同学习进度的学生）。
5. 如果用户提供了课程资料（docs），优先基于资料内容设计。
`,
	SkillQuizHelp: `
## 当前技能：测验与习题 (quiz_help)
**执行流程（请严格遵循）：**
若用户要求**出题**：
1. 先确认知识点范围、难度梯度（基础/提高/综合）、题目数量与题型。
2. 按要求生成题目，每题标注考查目标。
3. 单独输出参考答案与评分标准。

若用户要求**讲题**：
1. 先分析题目考查的核心知识点。
2. 给出清晰的分步解题过程。
3. 指出常见易错点，解释为何正确答案正确。
4. 提供 1 道同类型变式题供巩固。
`,
	SkillLessonSummary: `
## 当前技能：课堂总结 (lesson_summary)
**执行流程（请严格遵循）：**
1. 提炼本次课程的 3-5 个核心知识点，用简洁的标题列出。
2. 对每个知识点给出 1-2 句精炼解释。
3. 梳理知识点之间的逻辑关系（并列/递进/因果）。
4. 列出学生可能仍有疑惑的难点。
5. 给出课后复习建议和延伸阅读方向。
如果有课程资料（docs），以资料内容为基础进行总结。
`,
	SkillGeneral: `
## 当前技能：通用对话 (general)
按正常教学助手角色回复，用清晰、简洁、贴近课堂的语言。
`,
}

// AdvisorSystemPrompt 是 Advisor LLM 调用使用的系统 prompt
var AdvisorSystemPrompt = `你是教学助手的意图分析模块。
根据用户消息，从以下类型中选择最匹配的技能，并输出对本次任务的简短执行指引。

技能类型：
- student_qa：学生提问、概念理解、作业答疑、解题引导
- lesson_plan：教师备课、教学活动设计、课程目标规划
- quiz_help：出题、习题讲解、评分标准、考点分析
- lesson_summary：课堂总结、知识点梳理、学习回顾
- general：日常对话、系统问询、闲聊、其他

严格输出以下 JSON，不要有任何多余内容：
{"skill": "<技能类型>", "guidance": "<针对本次具体请求的1-2句执行提示，不超过60字>"}`
