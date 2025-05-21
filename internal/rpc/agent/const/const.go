package _const

const (
	//indexerNodes
	FileLoader       = "FileLoader"
	MarkdownSplitter = "MarkdownSplitter"
	RedisIndexer     = "RedisIndexer"

	//agentNodes
	InputToQuery   = "InputToQuery"
	ChatTemplate   = "ChatTemplate"
	ReactAgent     = "ReactAgent"
	RedisRetriever = "RedisRetriever"
	InputToHistory = "InputToHistory"

	//redis-stack中key前缀
	RedisPrefix = "eino:doc:"
	//redis-stack中向量索引名称
	IndexName = "vector_index"
	//向量维度(此处默认为doubao-embedding-large支持的最高4096)
	Dimension = 4096
	//存储原始文本内容，可用于关键词匹配
	ContentField = "content"
	//存储元信息，用于解释来源等
	MetadataField = "metadata"
	//保存向量化后的Content，用于向量相似度搜索
	VectorField = "content_vector"
	//深度
	DistanceField = "distance"
)
