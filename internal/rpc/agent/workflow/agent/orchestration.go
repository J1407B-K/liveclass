package agent

import (
	"context"
	_const "liveclass/internal/rpc/agent/const"
	"liveclass/internal/rpc/agent/memory"
	"liveclass/internal/rpc/agent/model"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

func BuildAgent(ctx context.Context, managers ...*memory.DBManager) (r compose.Runnable[*model.UserMessage, *schema.Message], err error) {
	g := compose.NewGraph[*model.UserMessage, *schema.Message]()

	if err = g.AddLambdaNode(
		_const.AdvisorNode,
		compose.InvokableLambdaWithOption(runAdvisor),
		compose.WithNodeName("AdvisorNode"),
	); err != nil {
		return nil, err
	}

	if err = g.AddLambdaNode(
		_const.InputToTemplateVars,
		compose.InvokableLambdaWithOption(newInputToTemplateVars),
		compose.WithNodeName("InputToTemplateVars"),
	); err != nil {
		return nil, err
	}

	//chatTemplate节点
	chatTemplateKey, err := newChatTemplate(ctx)
	if err != nil {
		return nil, err
	}
	if err = g.AddChatTemplateNode(
		_const.ChatTemplate,
		chatTemplateKey,
		compose.WithNodeName("ChatTemplate"),
	); err != nil {
		return nil, err
	}

	var dbm *memory.DBManager
	if len(managers) > 0 {
		dbm = managers[0]
	}
	reactAgentKey, err := newReactAgent(ctx, dbm)
	if err != nil {
		return nil, err
	}
	if err = g.AddLambdaNode(
		_const.ReactAgent,
		reactAgentKey,
		compose.WithNodeName("ReactAgent"),
	); err != nil {
		return nil, err
	}

	for _, e := range [][2]string{
		{compose.START, _const.AdvisorNode},
		{_const.AdvisorNode, _const.InputToTemplateVars},
		{_const.InputToTemplateVars, _const.ChatTemplate},
		{_const.ChatTemplate, _const.ReactAgent},
		{_const.ReactAgent, compose.END},
	} {
		if err = g.AddEdge(e[0], e[1]); err != nil {
			return nil, err
		}
	}

	r, err = g.Compile(ctx, compose.WithGraphName("Agent"), compose.WithNodeTriggerMode(compose.AllPredecessor))
	if err != nil {
		return nil, err
	}
	return r, nil
}
