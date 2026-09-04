package initialize

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"liveclass/internal/rpc/agent/dependency"
	"liveclass/internal/rpc/agent/global"
	"net/http"
	"time"

	"github.com/cloudwego/eino-ext/components/model/ark"
)

const defaultArkMultiModalEmbeddingURL = "https://ark.cn-beijing.volces.com/api/v3/embeddings/multimodal"

type MultiModalEmbedder struct {
	APIKey     string
	BaseURL    string
	Model      string
	Dimensions int
	Client     *http.Client
}

type MultiModalInputItem struct {
	Type     string         `json:"type"`
	Text     string         `json:"text,omitempty"`
	ImageURL *ImageURLField `json:"image_url,omitempty"`
}

type ImageURLField struct {
	URL string `json:"url"`
}

type multiModalEmbeddingRequest struct {
	Model          string                `json:"model"`
	Input          []MultiModalInputItem `json:"input"`
	Dimensions     int                   `json:"dimensions,omitempty"`
	EncodingFormat string                `json:"encoding_format,omitempty"`
}

type multiModalEmbeddingResponse struct {
	Created int64  `json:"created"`
	ID      string `json:"id"`
	Model   string `json:"model"`
	Object  string `json:"object"`

	Data struct {
		Embedding []float64 `json:"embedding"`
		Object    string    `json:"object"`
	} `json:"data"`

	Usage struct {
		PromptTokens        int `json:"prompt_tokens"`
		PromptTokensDetails struct {
			ImageTokens int `json:"image_tokens"`
			TextTokens  int `json:"text_tokens"`
		} `json:"prompt_tokens_details"`
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}

type arkErrorResponse struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Param     string `json:"param"`
	Type      string `json:"type"`
	RequestID string `json:"request_id"`
}

func InitChatModel(ctx context.Context) (cm *ark.ChatModel, err error) {
	//创建配置
	config := &ark.ChatModelConfig{
		Model:       global.Config.ChatModel,
		APIKey:      global.Config.APIKey,
		Temperature: &global.Config.ChatTemperature,
	}

	cm, err = ark.NewChatModel(ctx, config)
	if err != nil {
		return nil, err
	}
	return cm, nil
}

func InitMultiModalEmbedder(ctx context.Context) (*MultiModalEmbedder, error) {
	if global.Config.APIKey == "" {
		return nil, errors.New("empty ark api key")
	}
	if global.Config.EmbeddingModel == "" {
		return nil, errors.New("empty embedding model")
	}

	return &MultiModalEmbedder{
		APIKey:     global.Config.APIKey,
		BaseURL:    defaultArkMultiModalEmbeddingURL,
		Model:      global.Config.EmbeddingModel,
		Dimensions: 2048,
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

func (m *MultiModalEmbedder) EmbedText(ctx context.Context, text string) ([]float64, error) {
	if text == "" {
		return nil, errors.New("empty text")
	}

	vecs, err := m.EmbedTextBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(vecs) == 0 {
		return nil, errors.New("empty embeddings")
	}
	return vecs[0], nil
}

func (m *MultiModalEmbedder) EmbedTextBatch(ctx context.Context, texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, errors.New("empty texts")
	}

	result := make([][]float64, 0, len(texts))
	for _, text := range texts {
		items := []MultiModalInputItem{
			{
				Type: "text",
				Text: text,
			},
		}
		vec, err := m.embed(ctx, items)
		if err != nil {
			return nil, err
		}
		result = append(result, vec)
	}

	return result, nil
}

func (m *MultiModalEmbedder) EmbedMixed(ctx context.Context, items []MultiModalInputItem) ([]float64, error) {
	if len(items) == 0 {
		return nil, errors.New("empty multimodal input")
	}
	return m.embed(ctx, items)
}

func (m *MultiModalEmbedder) embed(ctx context.Context, items []MultiModalInputItem) ([]float64, error) {
	if m == nil {
		return nil, errors.New("nil multimodal embedder")
	}
	if m.APIKey == "" {
		return nil, errors.New("empty ark api key")
	}
	if m.BaseURL == "" {
		return nil, errors.New("empty base url")
	}
	if m.Model == "" {
		return nil, errors.New("empty model")
	}
	if m.Client == nil {
		return nil, errors.New("nil http client")
	}

	reqBody := multiModalEmbeddingRequest{
		Model:          m.Model,
		Input:          items,
		Dimensions:     m.Dimensions,
		EncodingFormat: "float",
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal multimodal embedding request failed: %w", err)
	}

	return dependency.Do(ctx, dependency.Embedding, "embed", func(callCtx context.Context) ([]float64, error) {
		return m.embedOnce(callCtx, bodyBytes)
	})
}

func (m *MultiModalEmbedder) embedOnce(ctx context.Context, bodyBytes []byte) ([]float64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.BaseURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("build multimodal embedding request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+m.APIKey)

	resp, err := m.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call ark multimodal embedding failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read ark multimodal embedding response failed: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var arkErr arkErrorResponse
		if err := json.Unmarshal(raw, &arkErr); err == nil && arkErr.Message != "" {
			return nil, &dependency.HTTPStatusError{StatusCode: resp.StatusCode, Body: fmt.Sprintf(
				"code=%s message=%s request_id=%s", arkErr.Code, arkErr.Message, arkErr.RequestID)}
		}
		return nil, &dependency.HTTPStatusError{StatusCode: resp.StatusCode, Body: string(raw)}
	}

	var parsed multiModalEmbeddingResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("unmarshal ark multimodal embedding response failed: %w, body=%s", err, string(raw))
	}

	if len(parsed.Data.Embedding) == 0 {
		return nil, fmt.Errorf("empty embedding data, body=%s", string(raw))
	}

	return parsed.Data.Embedding, nil
}
