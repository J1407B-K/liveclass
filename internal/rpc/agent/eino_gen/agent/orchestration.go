package indexer

import (
	"context"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	_const "liveclass/internal/rpc/agent/const"
	_type "liveclass/internal/rpc/agent/eino_gen/agent/type"
)

func BuildAgent(ctx context.Context) (r compose.Runnable[*_type.UserMessage, *schema.Message], err error) {
	g := compose.NewGraph[*_type.UserMessage, *schema.Message]()

	//inputToQuery节点
	_ = g.AddLambdaNode(_const.InputToQuery, compose.InvokableLambdaWithOption(newInputToQuery), compose.WithNodeName("InputToQuery"))

	//chatTemplate节点
	chatTemplateKeyOfChatTemplate, err := newChatTemplate(ctx)
	if err != nil {
		return nil, err
	}
	_ = g.AddChatTemplateNode(_const.ChatTemplate, chatTemplateKeyOfChatTemplate, compose.WithNodeName("ChatTemplate"))

	//reactAgent节点
	reactAgentKeyOfLambda, err := newReactAgent(ctx)
	if err != nil {
		return nil, err
	}
	_ = g.AddLambdaNode(_const.ReactAgent, reactAgentKeyOfLambda, compose.WithNodeName("ReactAgent"))

	//retriever节点
	redisRetrieverKeyOfRetriever, err := newRetriever(ctx)
	if err != nil {
		return nil, err
	}
	_ = g.AddRetrieverNode(_const.RedisRetriever, redisRetrieverKeyOfRetriever, compose.WithOutputKey("documents"))

	//inputToHistory节点
	_ = g.AddLambdaNode(_const.InputToHistory, compose.InvokableLambdaWithOption(newInputToHistory), compose.WithNodeName("UserMessageToVariables"))

	//START -> InputToQuery
	_ = g.AddEdge(compose.START, _const.InputToQuery)

	//START -> InputToHistory
	_ = g.AddEdge(compose.START, _const.InputToHistory)

	//inputToQuery -> retriever
	_ = g.AddEdge(_const.InputToQuery, _const.RedisRetriever)

	//retriever -> chatTemplate
	_ = g.AddEdge(_const.RedisRetriever, _const.ChatTemplate)

	//InputToHistory -> chatTemplate
	_ = g.AddEdge(_const.InputToHistory, _const.ChatTemplate)

	//chatTemplate -> reactAgent
	_ = g.AddEdge(_const.ChatTemplate, _const.ReactAgent)

	//reactAgent -> END
	_ = g.AddEdge(_const.ReactAgent, compose.END)

	r, err = g.Compile(ctx, compose.WithGraphName("Agent"), compose.WithNodeTriggerMode(compose.AllPredecessor))
	if err != nil {
		return nil, err
	}
	return r, nil
}
