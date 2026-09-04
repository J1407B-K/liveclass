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

	var dbm *memory.DBManager
	if len(managers) > 0 {
		dbm = managers[0]
	}
	adaptive, err := newAdaptiveExecutor(ctx, dbm)
	if err != nil {
		return nil, err
	}
	if err = g.AddLambdaNode(
		_const.ReactAgent,
		adaptive,
		compose.WithNodeName("AdaptivePlanExecutor"),
	); err != nil {
		return nil, err
	}

	for _, e := range [][2]string{
		{compose.START, _const.AdvisorNode},
		{_const.AdvisorNode, _const.ReactAgent},
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

func buildExecutionGraph(ctx context.Context, dbm *memory.DBManager, maxReActSteps int) (compose.Runnable[*model.UserMessage, *schema.Message], error) {
	g := compose.NewGraph[*model.UserMessage, *schema.Message]()
	if err := g.AddLambdaNode(_const.InputToTemplateVars, compose.InvokableLambdaWithOption(newInputToTemplateVars)); err != nil {
		return nil, err
	}
	template, err := newChatTemplate(ctx)
	if err != nil {
		return nil, err
	}
	if err = g.AddChatTemplateNode(_const.ChatTemplate, template); err != nil {
		return nil, err
	}
	reactNode, err := newReactAgentWithMaxSteps(ctx, dbm, maxReActSteps)
	if err != nil {
		return nil, err
	}
	if err = g.AddLambdaNode(_const.ReactAgent, reactNode); err != nil {
		return nil, err
	}
	for _, edge := range [][2]string{{compose.START, _const.InputToTemplateVars}, {_const.InputToTemplateVars, _const.ChatTemplate}, {_const.ChatTemplate, _const.ReactAgent}, {_const.ReactAgent, compose.END}} {
		if err = g.AddEdge(edge[0], edge[1]); err != nil {
			return nil, err
		}
	}
	return g.Compile(ctx, compose.WithGraphName("ReActExecutor"), compose.WithNodeTriggerMode(compose.AllPredecessor))
}
