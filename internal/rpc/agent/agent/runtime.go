package agent

import (
	"context"

	"liveclass/internal/rpc/agent/global"
	"liveclass/internal/rpc/agent/memory"
	"liveclass/internal/rpc/agent/rag"
	agentsession "liveclass/internal/rpc/agent/session"
)

// AgentRuntime owns deterministic orchestration around the single ReAct
// agent. Understanding, routing and tool selection remain model concerns;
// persistence, recovery, context selection and dependency policy do not.
type AgentRuntime struct {
	dbm            *memory.DBManager
	agentRunner    AgentRunner
	factRunner     FactRunner
	docRetriever   *rag.DocRetriever
	embedder       global.TextMultiModalEmbedder
	sessionManager *agentsession.Manager
}

func NewAgentRuntime(dbm *memory.DBManager, agentRunner AgentRunner, factRunner FactRunner, docRetriever *rag.DocRetriever, embedder global.TextMultiModalEmbedder, sessionManager *agentsession.Manager) *AgentRuntime {
	return &AgentRuntime{dbm: dbm, agentRunner: agentRunner, factRunner: factRunner, docRetriever: docRetriever, embedder: embedder, sessionManager: sessionManager}
}

func (r *AgentRuntime) Run(ctx context.Context, userID int64, sessionID, requestID, message string, lessonID int64) (string, error) {
	return ChatWithAgent(ctx, r.dbm, r.agentRunner, r.factRunner, r.docRetriever, r.embedder, r.sessionManager, userID, sessionID, requestID, message, lessonID)
}

// RunVariant exists only for reproducible offline evaluation. The v1 branch
// disables transcript recovery/compaction and uses the legacy flat retrieval
// path while retaining the same security and trace instrumentation.
func (r *AgentRuntime) RunVariant(ctx context.Context, variant string, userID int64, sessionID, requestID, message string, lessonID int64) (string, error) {
	if variant == "v1" {
		return ChatWithAgent(ctx, r.dbm, r.agentRunner, r.factRunner, r.docRetriever, r.embedder, nil, userID, sessionID, requestID, message, lessonID)
	}
	return r.Run(ctx, userID, sessionID, requestID, message, lessonID)
}
