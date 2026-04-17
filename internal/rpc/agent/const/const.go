package _const

const (
	//indexerNodes
	FileLoader       = "FileLoader"
	MarkdownSplitter = "MarkdownSplitter"
	RedisIndexer     = "RedisIndexer"

	//agentNodes
	AdvisorNode         = "AdvisorNode"
	InputToQuery        = "InputToQuery"
	InputToTemplateVars = "InputToTemplateVars"
	ChatTemplate        = "ChatTemplate"
	ReactAgent          = "ReactAgent"
	RedisRetriever      = "RedisRetriever"
	InputToHistory      = "InputToHistory"

	//最大重试次数
	MAXRETRY = 20

	FactInputToVars  = "FactInputToVars"
	FactChatTemplate = "FactChatTemplate"
	FactChatModel    = "FactChatModel"
	FactOutputParser = "FactOutputParser"
)
