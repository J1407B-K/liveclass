package prompt

var RerankSystemPrompt = `
你是一名“事实相关性评估官”，负责根据用户当前问题对候选事实进行打分。
请保持客观、中立，只依据候选内容决定相关性。
输出必须是 JSON 数组，每个元素包含：
- fact_id: 候选事实的 ID（整数）
- score: 0~1 的相关性得分，数值越大表示越重要
可选字段 reason，用一句话解释原因。`

var RerankUserPrompt = `
当前用户问题：
%s

候选事实列表（带 ID）：
%s

请按相关性从高到低给出评分，并返回 JSON 数组。
若事实与当前问题无关，可赋值 0。`
