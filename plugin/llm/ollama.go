package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	storepb "github.com/usememos/memos/proto/gen/store"
)

const (
	ollamaDefaultHost           = "http://localhost:11434"
	ollamaDefaultModel          = "llama3.2"
	ollamaDefaultEmbeddingModel = "nomic-embed-text"
)

// OllamaProvider implements the Provider interface for Ollama.
type OllamaProvider struct {
	*BaseProvider
	host           string
	defaultModel   string
	embeddingModel string
}

// NewOllamaProvider creates a new Ollama provider.
func NewOllamaProvider(config *ProviderConfig) *OllamaProvider {
	host := ollamaDefaultHost
	defaultModel := ollamaDefaultModel
	embeddingModel := ollamaDefaultEmbeddingModel

	if config.OllamaHost != "" {
		host = config.OllamaHost
	}
	if config.DefaultModel != "" {
		defaultModel = config.DefaultModel
	}
	if config.EmbeddingModel != "" {
		embeddingModel = config.EmbeddingModel
	}

	return &OllamaProvider{
		BaseProvider:   NewBaseProvider(config),
		host:           host,
		defaultModel:   defaultModel,
		embeddingModel: embeddingModel,
	}
}

// NewOllamaProviderFromProto creates a new Ollama provider from proto config.
func NewOllamaProviderFromProto(pbConfig *storepb.LLMOllamaConfig) *OllamaProvider {
	config := &ProviderConfig{
		Type:           ProviderOllama,
		OllamaHost:     pbConfig.GetHost(),
		DefaultModel:   pbConfig.GetDefaultModel(),
		EmbeddingModel: pbConfig.GetEmbeddingModel(),
	}
	return NewOllamaProvider(config)
}

// GetType returns the provider type.
func (p *OllamaProvider) GetType() ProviderType {
	return ProviderOllama
}

// GetName returns the display name.
func (p *OllamaProvider) GetName() string {
	return "Ollama"
}

// IsConfigured checks if the provider is properly configured.
// Ollama doesn't require an API key, so we just check if host is set.
func (p *OllamaProvider) IsConfigured(ctx context.Context) bool {
	return p.host != ""
}

// GetDefaultModel returns the default model.
func (p *OllamaProvider) GetDefaultModel() string {
	return p.defaultModel
}

// GetAvailableModels returns available models from the Ollama server.
func (p *OllamaProvider) GetAvailableModels(ctx context.Context) ([]string, error) {
	if !p.IsConfigured(ctx) {
		return nil, ErrProviderNotConfigured
	}

	url := fmt.Sprintf("%s/api/tags", p.host)

	respBody, err := p.DoRequest(ctx, http.MethodGet, url, nil, nil)
	if err != nil {
		return nil, err
	}

	var resp ollamaModelsResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse models response: %w", err)
	}

	models := make([]string, len(resp.Models))
	for i, m := range resp.Models {
		models[i] = m.Name
	}

	return models, nil
}

// Complete performs chat completion using Ollama's API.
func (p *OllamaProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	if !p.IsConfigured(ctx) {
		return nil, ErrProviderNotConfigured
	}

	model := req.Model
	if model == "" {
		model = p.defaultModel
	}

	// Build Ollama request
	messages := make([]ollamaMessage, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = ollamaMessage{
			Role:    string(m.Role),
			Content: m.Content,
		}
	}

	ollamaReq := ollamaChatRequest{
		Model:    model,
		Messages: messages,
		Stream:   false, // We don't support streaming yet
	}

	// Add options if specified
	if req.Temperature > 0 || req.TopP > 0 {
		ollamaReq.Options = &ollamaOptions{}
		if req.Temperature > 0 {
			ollamaReq.Options.Temperature = req.Temperature
		}
		if req.TopP > 0 {
			ollamaReq.Options.TopP = req.TopP
		}
	}

	url := fmt.Sprintf("%s/api/chat", p.host)

	respBody, err := p.DoRequest(ctx, http.MethodPost, url, ollamaReq, nil)
	if err != nil {
		return nil, err
	}

	var resp ollamaChatResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse completion response: %w", err)
	}

	return &CompletionResponse{
		Content: resp.Message.Content,
		Model:   resp.Model,
		Usage: &TokenUsage{
			PromptTokens:     resp.PromptEvalCount,
			CompletionTokens: resp.EvalCount,
			TotalTokens:      resp.PromptEvalCount + resp.EvalCount,
		},
	}, nil
}

// Embed generates embeddings using Ollama's API.
func (p *OllamaProvider) Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	if !p.IsConfigured(ctx) {
		return nil, ErrProviderNotConfigured
	}

	model := req.Model
	if model == "" {
		model = p.embeddingModel
	}

	// Ollama embedding API takes one input at a time, so we need to make multiple requests
	embeddings := make([][]float32, len(req.Input))
	var totalTokens int

	for i, input := range req.Input {
		ollamaReq := ollamaEmbedRequest{
			Model: model,
			Input: input,
		}

		url := fmt.Sprintf("%s/api/embed", p.host)

		respBody, err := p.DoRequest(ctx, http.MethodPost, url, ollamaReq, nil)
		if err != nil {
			return nil, err
		}

		var resp ollamaEmbedResponse
		if err := json.Unmarshal(respBody, &resp); err != nil {
			return nil, fmt.Errorf("failed to parse embedding response: %w", err)
		}

		if len(resp.Embeddings) > 0 {
			embeddings[i] = resp.Embeddings[0]
		}
		totalTokens += resp.PromptEvalCount
	}

	return &EmbeddingResponse{
		Embeddings: embeddings,
		Model:      model,
		Usage: &TokenUsage{
			PromptTokens: totalTokens,
			TotalTokens:  totalTokens,
		},
	}, nil
}

// SuggestTags suggests tags for the given content.
func (p *OllamaProvider) SuggestTags(ctx context.Context, req *SuggestTagsRequest) (*SuggestTagsResponse, error) {
	return p.DefaultSuggestTags(ctx, p, req)
}

// Summarize generates a summary of the given content.
func (p *OllamaProvider) Summarize(ctx context.Context, req *SummarizeRequest) (*SummarizeResponse, error) {
	return p.DefaultSummarize(ctx, p, req)
}

// ToProto converts the provider configuration to proto format.
func (p *OllamaProvider) ToProto() *storepb.LLMOllamaConfig {
	return &storepb.LLMOllamaConfig{
		Host:           p.host,
		DefaultModel:   p.defaultModel,
		EmbeddingModel: p.embeddingModel,
	}
}

// CheckHealth verifies the Ollama server is reachable.
func (p *OllamaProvider) CheckHealth(ctx context.Context) error {
	if !p.IsConfigured(ctx) {
		return ErrProviderNotConfigured
	}

	// Ollama has a simple root endpoint that returns version info
	url := fmt.Sprintf("%s/api/version", p.host)

	_, err := p.DoRequest(ctx, http.MethodGet, url, nil, nil)
	if err != nil {
		return fmt.Errorf("ollama health check failed: %w", err)
	}

	return nil
}

// Ollama API request/response types

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaOptions struct {
	Temperature float64 `json:"temperature,omitempty"`
	TopP        float64 `json:"top_p,omitempty"`
	NumPredict  int     `json:"num_predict,omitempty"`
}

type ollamaChatRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Options  *ollamaOptions  `json:"options,omitempty"`
}

type ollamaChatResponse struct {
	Model   string `json:"model"`
	Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
	Done            bool   `json:"done"`
	DoneReason      string `json:"done_reason,omitempty"`
	TotalDuration   int64  `json:"total_duration,omitempty"`
	LoadDuration    int64  `json:"load_duration,omitempty"`
	PromptEvalCount int    `json:"prompt_eval_count,omitempty"`
	EvalCount       int    `json:"eval_count,omitempty"`
}

type ollamaEmbedRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type ollamaEmbedResponse struct {
	Model           string      `json:"model"`
	Embeddings      [][]float32 `json:"embeddings"`
	TotalDuration   int64       `json:"total_duration,omitempty"`
	LoadDuration    int64       `json:"load_duration,omitempty"`
	PromptEvalCount int         `json:"prompt_eval_count,omitempty"`
}

type ollamaModelsResponse struct {
	Models []struct {
		Name       string `json:"name"`
		Model      string `json:"model"`
		ModifiedAt string `json:"modified_at"`
		Size       int64  `json:"size"`
		Digest     string `json:"digest"`
		Details    struct {
			Format            string   `json:"format"`
			Family            string   `json:"family"`
			Families          []string `json:"families"`
			ParameterSize     string   `json:"parameter_size"`
			QuantizationLevel string   `json:"quantization_level"`
		} `json:"details"`
	} `json:"models"`
}
