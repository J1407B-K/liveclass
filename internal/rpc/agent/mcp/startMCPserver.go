package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func StartMCPServer() {
	svr := server.NewMCPServer("mcp-server", mcp.LATEST_PROTOCOL_VERSION)

	// 计数器
	svr.AddTool(mcp.NewTool("calculate",
		mcp.WithDescription("Perform basic arithmetic operations"),
		mcp.WithString("operation",
			mcp.Required(),
			mcp.Description("Operation: add, subtract, multiply, divide"),
			mcp.Enum("add", "subtract", "multiply", "divide"),
		),
		mcp.WithNumber("x", mcp.Required(), mcp.Description("First number")),
		mcp.WithNumber("y", mcp.Required(), mcp.Description("Second number")),
	), calculateHandler)

	// 哈希加密
	svr.AddTool(mcp.NewTool("sha256",
		mcp.WithDescription("Compute SHA-256 hash of input string"),
		mcp.WithString("input", mcp.Required(), mcp.Description("String to hash")),
	), sha256Handler)

	// 生成uuid
	svr.AddTool(mcp.NewTool("uuid4",
		mcp.WithDescription("Generate a random UUID (version 4)"),
	), uuidHandler)

	// 随机数
	svr.AddTool(mcp.NewTool("random_number",
		mcp.WithDescription("Generate a random integer between min and max (inclusive)"),
		mcp.WithNumber("min", mcp.Required(), mcp.Description("Minimum value")),
		mcp.WithNumber("max", mcp.Required(), mcp.Description("Maximum value")),
	), randomNumberHandler)

	// 获取当前时间
	svr.AddTool(mcp.NewTool("current_time",
		mcp.WithDescription("Return current server time in RFC3339 format"),
	), timeHandler)

	go func() {
		log.Println("Start MCP server on http://localhost:12345")
		if err := server.NewSSEServer(svr, server.WithBaseURL("http://localhost:12345")).Start("localhost:12345"); err != nil {
			log.Fatal(err)
		}
	}()
}

func calculateHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.Params.Arguments.(map[string]any)
	op := args["operation"].(string)
	x := args["x"].(float64)
	y := args["y"].(float64)
	var res float64
	switch op {
	case "add":
		res = x + y
	case "subtract":
		res = x - y
	case "multiply":
		res = x * y
	case "divide":
		if y == 0 {
			return mcp.NewToolResultText("Cannot divide by zero"), nil
		}
		res = x / y
	}
	return mcp.NewToolResultText(fmt.Sprintf("%.2f", res)), nil
}

func sha256Handler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input := request.Params.Arguments.(map[string]any)["input"].(string)
	h := sha256.Sum256([]byte(input))
	hstr := hex.EncodeToString(h[:])
	return mcp.NewToolResultText(hstr), nil
}

func uuidHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := uuid.New().String()
	return mcp.NewToolResultText(id), nil
}

func randomNumberHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.Params.Arguments.(map[string]any)
	min := int(args["min"].(float64))
	max := int(args["max"].(float64))
	if min > max {
		min, max = max, min
	}
	n := rand.Intn(max-min+1) + min
	return mcp.NewToolResultText(fmt.Sprintf("%d", n)), nil
}

func timeHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	now := time.Now().Format(time.RFC3339)
	return mcp.NewToolResultText(now), nil
}
