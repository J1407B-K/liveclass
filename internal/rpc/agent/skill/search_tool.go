package skill

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

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
	url := fmt.Sprintf(
		"http://openserp:7000/mega/search?text=%s&limit=5",
		query,
	)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var results []SearchResult
	err = json.NewDecoder(resp.Body).Decode(&results)
	if err != nil {
		return nil, err
	}

	return results, nil
}
