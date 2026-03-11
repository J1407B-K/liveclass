package fact

import (
	"context"
	_const "liveclass/internal/rpc/agent/const"
	"liveclass/internal/rpc/agent/global"
	"liveclass/internal/rpc/agent/model"

	"github.com/cloudwego/eino/compose"
)

func BuildFactExtractor(ctx context.Context) (compose.Runnable[*model.FactExtractInput, []*model.FactCandidate], error) {
	g := compose.NewGraph[*model.FactExtractInput, []*model.FactCandidate]()

	_ = g.AddLambdaNode(
		_const.FactInputToVars,
		compose.InvokableLambdaWithOption(factInputToVars),
		compose.WithNodeName("FactInputToVars"),
	)

	ctp, err := newFactExtractTemplate(ctx)
	if err != nil {
		return nil, err
	}
	_ = g.AddChatTemplateNode(
		_const.FactChatTemplate,
		ctp,
		compose.WithNodeName("FactChatTemplate"),
	)

	_ = g.AddChatModelNode(
		_const.FactChatModel,
		global.ChatModel,
		compose.WithNodeName("FactChatModel"),
	)

	_ = g.AddLambdaNode(
		_const.FactOutputParser,
		compose.InvokableLambdaWithOption(factMessageToCandidates),
		compose.WithNodeName("FactOutputParser"),
	)

	_ = g.AddEdge(compose.START, _const.FactInputToVars)
	_ = g.AddEdge(_const.FactInputToVars, _const.FactChatTemplate)
	_ = g.AddEdge(_const.FactChatTemplate, _const.FactChatModel)
	_ = g.AddEdge(_const.FactChatModel, _const.FactOutputParser)
	_ = g.AddEdge(_const.FactOutputParser, compose.END)

	r, err := g.Compile(
		ctx,
		compose.WithGraphName("FactExtractor"),
		compose.WithNodeTriggerMode(compose.AllPredecessor),
	)
	if err != nil {
		return nil, err
	}

	return r, nil
}
