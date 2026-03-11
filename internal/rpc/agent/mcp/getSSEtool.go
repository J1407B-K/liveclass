package mcp

import (
	"context"
	"fmt"
	"log"

	mcpp "github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/cloudwego/eino/components/tool"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

func GetSSETool(ctx context.Context, url string) ([]tool.BaseTool, error) {
	log.Printf("connecting MCP SSE: %s", url)

	cli, err := client.NewSSEMCPClient(url)
	if err != nil {
		return nil, fmt.Errorf("NewSSEMCPClient failed: %w", err)
	}

	if err := cli.Start(ctx); err != nil {
		return nil, fmt.Errorf("cli.Start failed: %w", err)
	}

	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "example-client",
		Version: "1.0.0",
	}

	if _, err := cli.Initialize(ctx, initRequest); err != nil {
		return nil, fmt.Errorf("cli.Initialize failed: %w", err)
	}

	tools, err := mcpp.GetTools(ctx, &mcpp.Config{Cli: cli})
	if err != nil {
		return nil, fmt.Errorf("mcpp.GetTools failed: %w", err)
	}

	log.Println("GetSSETool success")
	return tools, nil
}
