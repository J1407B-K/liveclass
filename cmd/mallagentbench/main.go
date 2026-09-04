package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/kitex/client"
	"gorm.io/gorm/clause"

	"liveclass/idl/kitex_gen/mall/mallservice"
	"liveclass/internal/rpc/agent/dependency"
	"liveclass/internal/rpc/agent/global"
	"liveclass/internal/rpc/agent/initialize"
	agenttool "liveclass/internal/rpc/agent/tool"
	"liveclass/internal/rpc/agent/toolruntime"
	"liveclass/internal/rpc/mall/domain"
)

type result struct {
	GeneratedAt string           `json:"generated_at"`
	Scenario    string           `json:"scenario"`
	Assertions  map[string]bool  `json:"assertions"`
	OrderID     string           `json:"order_id"`
	ReplayID    string           `json:"replay_order_id"`
	FinalState  map[string]int64 `json:"final_state"`
}

func main() {
	var output, address string
	flag.StringVar(&output, "output", "benchmark-results/mall-agent-tool-security.json", "result JSON path")
	flag.StringVar(&address, "address", "127.0.0.1:9010", "Mall Kitex address")
	flag.Parse()
	initialize.SetupViper()
	must(dependency.Configure(global.Config.Resilience))

	db, _, err := domain.OpenMySQL()
	must(err)
	must(domain.MigrateOrder(db))
	must(domain.MigrateInventory(db))
	must(domain.MigratePoints(db))
	now := time.Now().UnixNano()
	userID := int64(7_700_000_000) + now%1_000_000
	productID := int64(9_700_000_000) + now%1_000_000
	must(db.Create(&domain.Product{ID: productID, Name: "Agent Security Benchmark", Description: "two-turn confirmation", PointsPrice: 100, Active: true}).Error)
	must(db.Create(&domain.Inventory{ProductID: productID, Available: 2, Version: 1}).Error)
	must(db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&domain.PointsAccount{UserID: userID, Balance: 1000, Version: 1}).Error)

	cli, err := mallservice.NewClient("mallservice", client.WithHostPorts(address))
	must(err)
	_, prepare, exchange, err := agenttool.NewMallTools(cli, []byte("12345678901234567890123456789012"))
	must(err)

	firstTurn := toolruntime.WithPrincipal(context.Background(), toolruntime.Principal{UserID: userID, SessionID: "mall-agent-bench", RequestID: "turn-1", ApprovedTools: map[string]bool{"exchange_mall_product": true}})
	preparedJSON, err := prepare.InvokableRun(firstTurn, fmt.Sprintf(`{"product_id":%d,"quantity":1}`, productID))
	must(err)
	var prepared agenttool.MallPrepareResponse
	must(json.Unmarshal([]byte(preparedJSON), &prepared))

	registry := toolruntime.NewRegistry(nil)
	must(registry.Register(context.Background(), exchange, toolruntime.ToolSpec{Name: "exchange_mall_product", Permission: toolruntime.PermissionAuthenticated, RiskLevel: toolruntime.RiskHigh, Timeout: 20 * time.Second, Retry: toolruntime.RetryPolicy{Attempts: 1}}))
	wrapped := registry.Tools()[0].(tool.InvokableTool)
	args, _ := json.Marshal(agenttool.MallExchangeRequest{ConfirmationToken: prepared.ConfirmationToken})
	_, sameTurnErr := wrapped.InvokableRun(firstTurn, string(args))

	secondTurnDenied := toolruntime.WithPrincipal(context.Background(), toolruntime.Principal{UserID: userID, SessionID: "mall-agent-bench", RequestID: "turn-2"})
	_, noApprovalErr := wrapped.InvokableRun(secondTurnDenied, string(args))
	secondTurnApproved := toolruntime.WithPrincipal(context.Background(), toolruntime.Principal{UserID: userID, SessionID: "mall-agent-bench", RequestID: "turn-2", ApprovedTools: map[string]bool{"exchange_mall_product": true}})
	tamperedArgs, _ := json.Marshal(agenttool.MallExchangeRequest{ConfirmationToken: prepared.ConfirmationToken + "x"})
	_, tamperedErr := wrapped.InvokableRun(secondTurnApproved, string(tamperedArgs))

	exchangedJSON, err := wrapped.InvokableRun(secondTurnApproved, string(args))
	must(err)
	var exchanged agenttool.MallExchangeResponse
	must(json.Unmarshal([]byte(exchangedJSON), &exchanged))
	replayCtx := toolruntime.WithPrincipal(context.Background(), toolruntime.Principal{UserID: userID, SessionID: "mall-agent-bench", RequestID: "turn-3", ApprovedTools: map[string]bool{"exchange_mall_product": true}})
	replayedJSON, err := wrapped.InvokableRun(replayCtx, string(args))
	must(err)
	var replayed agenttool.MallExchangeResponse
	must(json.Unmarshal([]byte(replayedJSON), &replayed))

	var inventory domain.Inventory
	var account domain.PointsAccount
	must(db.First(&inventory, "product_id = ?", productID).Error)
	must(db.First(&account, "user_id = ?", userID).Error)
	out := result{GeneratedAt: time.Now().UTC().Format(time.RFC3339Nano), Scenario: "agent_two_turn_confirmation", OrderID: exchanged.OrderID, ReplayID: replayed.OrderID,
		Assertions: map[string]bool{"same_turn_rejected": sameTurnErr != nil, "missing_explicit_approval_rejected": noApprovalErr != nil, "tampered_token_rejected": tamperedErr != nil, "approved_next_turn_confirmed": exchanged.Status == domain.OrderConfirmed, "replay_returns_same_order": exchanged.OrderID != "" && exchanged.OrderID == replayed.OrderID},
		FinalState: map[string]int64{"available": inventory.Available, "reserved": inventory.Reserved, "sold": inventory.Sold, "points_balance": account.Balance}}
	raw, err := json.MarshalIndent(out, "", "  ")
	must(err)
	must(os.WriteFile(output, append(raw, '\n'), 0o644))
	fmt.Println(string(raw))
	for name, passed := range out.Assertions {
		if !passed {
			panic("assertion failed: " + name)
		}
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
