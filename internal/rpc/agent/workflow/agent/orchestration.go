package agent

import (
	"context"
	_const "liveclass/internal/rpc/agent/const"
	"liveclass/internal/rpc/agent/model"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

func BuildAgent(ctx context.Context) (r compose.Runnable[*model.UserMessage, *schema.Message], err error) {
	g := compose.NewGraph[*model.UserMessage, *schema.Message]()

	//inputToTemplateVars节点
	_ = g.AddLambdaNode(
		_const.InputToTemplateVars,
		compose.InvokableLambdaWithOption(newInputToTemplateVars),
		compose.WithNodeName("InputToTemplateVars"),
	)

	//chatTemplate节点
	chatTemplateKey, err := newChatTemplate(ctx)
	if err != nil {
		return nil, err
	}
	_ = g.AddChatTemplateNode(
		_const.ChatTemplate,
		chatTemplateKey,
		compose.WithNodeName("ChatTemplate"),
	)

	//reactAgent节点
	reactAgentKey, err := newReactAgent(ctx)
	if err != nil {
		return nil, err
	}
	_ = g.AddLambdaNode(
		_const.ReactAgent,
		reactAgentKey,
		compose.WithNodeName("ReactAgent"),
	)

	_ = g.AddEdge(compose.START, _const.InputToTemplateVars)
	_ = g.AddEdge(_const.InputToTemplateVars, _const.ChatTemplate)
	_ = g.AddEdge(_const.ChatTemplate, _const.ReactAgent)
	_ = g.AddEdge(_const.ReactAgent, compose.END)

	r, err = g.Compile(ctx, compose.WithGraphName("Agent"), compose.WithNodeTriggerMode(compose.AllPredecessor))
	if err != nil {
		return nil, err
	}
	return r, nil
}
