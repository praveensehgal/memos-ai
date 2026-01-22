package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	storepb "github.com/usememos/memos/proto/gen/store"
)

const (
	openAIBaseURL        = "https://api.openai.com/v1"
	openAIDefaultModel   = "gpt-4o-mini"
	openAIEmbeddingModel = "text-embedding-3-small"
)

// OpenAIProvider implements the Provider interface for OpenAI.
type OpenAIProvider struct {
	*BaseProvider
	apiKey         string
	baseURL        string
	defaultModel   string
	embeddingModel string
}

// NewOpenAIProvider creates a new OpenAI provider.
func NewOpenAIProvider(config *ProviderConfig) *OpenAIProvider {
	baseURL := openAIBaseURL
	defaultModel := openAIDefaultModel
	embeddingModel := openAIEmbeddingModel

	if config.BaseURL != "" {
		baseURL = config.BaseURL
	}
	if config.DefaultModel != "" {
		defaultModel = config.DefaultModel
	}
	if config.EmbeddingModel != "" {
		embeddingModel = config.EmbeddingModel
	}

	return &OpenAIProvider{
		BaseProvider:   NewBaseProvider(config),
		apiKey:         config.APIKey,
		baseURL:        baseURL,
		defaultModel:   defaultModel,
		embeddingModel: embeddingModel,
	}
}

// NewOpenAIProviderFromProto creates a new OpenAI provider from proto config.
func NewOpenAIProviderFromProto(pbConfig *storepb.LLMOpenAIConfig) *OpenAIProvider {
	config := &ProviderConfig{
		Type:           ProviderOpenAI,
		APIKey:         pbConfig.GetApiKey(),
		BaseURL:        pbConfig.GetBaseUrl(),
		DefaultModel:   pbConfig.GetDefaultModel(),
		EmbeddingModel: pbConfig.GetEmbeddingModel(),
	}
	return NewOpenAIProvider(config)
}

// GetType returns the provider type.
func (p *OpenAIProvider) GetType() ProviderType {
	return ProviderOpenAI
}

// GetName returns the display name.
func (p *OpenAIProvider) GetName() string {
	return "OpenAI"
}

// IsConfigured checks if the provider is properly configured.
func (p *OpenAIProvider) IsConfigured(ctx context.Context) bool {
	return p.apiKey != ""
}

// GetDefaultModel returns the default model.
func (p *OpenAIProvider) GetDefaultModel() string {
	return p.defaultModel
}

// GetAvailableModels returns available models.
func (p *OpenAIProvider) GetAvailableModels(ctx context.Context) ([]string, error) {
	if !p.IsConfigured(ctx) {
		return nil, ErrProviderNotConfigured
	}

	url := fmt.Sprintf("%s/models", p.baseURL)
	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", p.apiKey),
	}

	respBody, err := p.DoRequest(ctx, http.MethodGet, url, nil, headers)
	if err != nil {
		return nil, err
	}

	var resp openAIModelsResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse models response: %w", err)
	}

	// Filter to only chat models
	var models []string
	for _, m := range resp.Data {
		// Include GPT models and o1 models
		if isOpenAIChatModel(m.ID) {
			models = append(models, m.ID)
		}
	}

	return models, nil
}

// Complete performs chat completion.
func (p *OpenAIProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	if !p.IsConfigured(ctx) {
		return nil, ErrProviderNotConfigured
	}

	model := req.Model
	if model == "" {
		model = p.defaultModel
	}

	// Build OpenAI request
	messages := make([]openAIMessage, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = openAIMessage{
			Role:    string(m.Role),
			Content: m.Content,
		}
	}

	openAIReq := openAIChatRequest{
		Model:    model,
		Messages: messages,
	}

	if req.MaxTokens > 0 {
		openAIReq.MaxTokens = req.MaxTokens
	}
	if req.Temperature > 0 {
		openAIReq.Temperature = req.Temperature
	}
	if req.TopP > 0 {
		openAIReq.TopP = req.TopP
	}

	url := fmt.Sprintf("%s/chat/completions", p.baseURL)
	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", p.apiKey),
	}

	respBody, err := p.DoRequest(ctx, http.MethodPost, url, openAIReq, headers)
	if err != nil {
		return nil, err
	}

	var resp openAIChatResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse completion response: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no completion choices returned")
	}

	return &CompletionResponse{
		Content: resp.Choices[0].Message.Content,
		Model:   resp.Model,
		Usage: &TokenUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}, nil
}

// Embed generates embeddings for the given input.
func (p *OpenAIProvider) Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	if !p.IsConfigured(ctx) {
		return nil, ErrProviderNotConfigured
	}

	model := req.Model
	if model == "" {
		model = p.embeddingModel
	}

	openAIReq := openAIEmbeddingRequest{
		Model: model,
		Input: req.Input,
	}

	url := fmt.Sprintf("%s/embeddings", p.baseURL)
	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", p.apiKey),
	}

	respBody, err := p.DoRequest(ctx, http.MethodPost, url, openAIReq, headers)
	if err != nil {
		return nil, err
	}

	var resp openAIEmbeddingResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse embedding response: %w", err)
	}

	embeddings := make([][]float32, len(resp.Data))
	for i, d := range resp.Data {
		embeddings[i] = d.Embedding
	}

	return &EmbeddingResponse{
		Embeddings: embeddings,
		Model:      resp.Model,
		Usage: &TokenUsage{
			PromptTokens: resp.Usage.PromptTokens,
			TotalTokens:  resp.Usage.TotalTokens,
		},
	}, nil
}

// SuggestTags suggests tags for the given content.
func (p *OpenAIProvider) SuggestTags(ctx context.Context, req *SuggestTagsRequest) (*SuggestTagsResponse, error) {
	return p.DefaultSuggestTags(ctx, p, req)
}

// Summarize generates a summary of the given content.
func (p *OpenAIProvider) Summarize(ctx context.Context, req *SummarizeRequest) (*SummarizeResponse, error) {
	return p.DefaultSummarize(ctx, p, req)
}

// isOpenAIChatModel checks if a model ID is a chat model.
func isOpenAIChatModel(id string) bool {
	prefixes := []string{"gpt-4", "gpt-3.5", "o1", "chatgpt"}
	for _, prefix := range prefixes {
		if len(id) >= len(prefix) && id[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

// OpenAI API request/response types

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	TopP        float64         `json:"top_p,omitempty"`
}

type openAIChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type openAIEmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type openAIEmbeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

type openAIModelsResponse struct {
	Object string `json:"object"`
	Data   []struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		OwnedBy string `json:"owned_by"`
	} `json:"data"`
}
