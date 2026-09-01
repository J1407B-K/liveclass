package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"liveclass/internal/rpc/agent/dependency"
	"liveclass/internal/rpc/agent/global"
	"net/http"
	"net/url"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type SearchResult struct {
	Rank        int    `json:"rank"`
	URL         string `json:"url"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Engine      string `json:"engine"`
}

func NewSearchTool() (tool.BaseTool, error) {
	return utils.InferTool("query_on_internet", "在网络上搜索用户咨询相关问题", SearchWeb)
}

func SearchWeb(ctx context.Context, query string) ([]SearchResult, error) {
	results, err := dependency.Do(ctx, dependency.WebSearch, "search", func(callCtx context.Context) ([]SearchResult, error) {
		return searchWebOnce(callCtx, query)
	})
	if err != nil {
		dependency.Fallback(dependency.WebSearch, "search")
		return []SearchResult{}, nil
	}
	return results, nil
}

func searchWebOnce(ctx context.Context, query string) ([]SearchResult, error) {
	baseURL := "http://openserp:7000/mega/search"
	if global.Config != nil && strings.TrimSpace(global.Config.WebSearchURL) != "" {
		baseURL = strings.TrimSpace(global.Config.WebSearchURL)
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse web search URL: %w", err)
	}
	params := u.Query()
	params.Set("text", query)
	params.Set("limit", "5")
	u.RawQuery = params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, &dependency.HTTPStatusError{StatusCode: resp.StatusCode, Body: string(body)}
	}

	var results []SearchResult
	err = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&results)
	if err != nil {
		return nil, err
	}

	return results, nil
}
