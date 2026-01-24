package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	storepb "github.com/usememos/memos/proto/gen/store"
)

const (
	anthropicBaseURL      = "https://api.anthropic.com"
	anthropicDefaultModel = "claude-3-haiku-20240307"
	anthropicAPIVersion   = "2023-06-01"
)

// AnthropicProvider implements the Provider interface for Anthropic Claude.
type AnthropicProvider struct {
	*BaseProvider
	apiKey       string
	baseURL      string
	defaultModel string
}

// NewAnthropicProvider creates a new Anthropic provider.
func NewAnthropicProvider(config *ProviderConfig) *AnthropicProvider {
	baseURL := anthropicBaseURL
	defaultModel := anthropicDefaultModel

	if config.BaseURL != "" {
		baseURL = config.BaseURL
	}
	if config.DefaultModel != "" {
		defaultModel = config.DefaultModel
	}

	return &AnthropicProvider{
		BaseProvider: NewBaseProvider(config),
		apiKey:       config.APIKey,
		baseURL:      baseURL,
		defaultModel: defaultModel,
	}
}

// NewAnthropicProviderFromProto creates a new Anthropic provider from proto config.
func NewAnthropicProviderFromProto(pbConfig *storepb.LLMAnthropicConfig) *AnthropicProvider {
	config := &ProviderConfig{
		Type:         ProviderAnthropic,
		APIKey:       pbConfig.GetApiKey(),
		BaseURL:      pbConfig.GetBaseUrl(),
		DefaultModel: pbConfig.GetDefaultModel(),
	}
	return NewAnthropicProvider(config)
}

// GetType returns the provider type.
func (p *AnthropicProvider) GetType() ProviderType {
	return ProviderAnthropic
}

// GetName returns the display name.
func (p *AnthropicProvider) GetName() string {
	return "Anthropic"
}

// IsConfigured checks if the provider is properly configured.
func (p *AnthropicProvider) IsConfigured(ctx context.Context) bool {
	return p.apiKey != ""
}

// GetDefaultModel returns the default model.
func (p *AnthropicProvider) GetDefaultModel() string {
	return p.defaultModel
}

// GetAvailableModels returns available models.
func (p *AnthropicProvider) GetAvailableModels(ctx context.Context) ([]string, error) {
	if !p.IsConfigured(ctx) {
		return nil, ErrProviderNotConfigured
	}

	// Anthropic doesn't have a models endpoint, return known models
	return []string{
		"claude-3-5-sonnet-20241022",
		"claude-3-5-haiku-20241022",
		"claude-3-opus-20240229",
		"claude-3-sonnet-20240229",
		"claude-3-haiku-20240307",
	}, nil
}

// Complete performs chat completion.
func (p *AnthropicProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	if !p.IsConfigured(ctx) {
		return nil, ErrProviderNotConfigured
	}

	model := req.Model
	if model == "" {
		model = p.defaultModel
	}

	// Build Anthropic request - extract system message separately
	var system string
	messages := make([]anthropicMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		if m.Role == RoleSystem {
			system = m.Content
		} else {
			messages = append(messages, anthropicMessage{
				Role:    string(m.Role),
				Content: m.Content,
			})
		}
	}

	anthropicReq := anthropicMessagesRequest{
		Model:     model,
		Messages:  messages,
		MaxTokens: 4096, // Anthropic requires max_tokens
	}

	if system != "" {
		anthropicReq.System = system
	}
	if req.MaxTokens > 0 {
		anthropicReq.MaxTokens = req.MaxTokens
	}
	if req.Temperature > 0 {
		anthropicReq.Temperature = req.Temperature
	}
	if req.TopP > 0 {
		anthropicReq.TopP = req.TopP
	}

	url := fmt.Sprintf("%s/v1/messages", p.baseURL)
	headers := map[string]string{
		"x-api-key":         p.apiKey,
		"anthropic-version": anthropicAPIVersion,
	}

	respBody, err := p.DoRequest(ctx, http.MethodPost, url, anthropicReq, headers)
	if err != nil {
		return nil, err
	}

	var resp anthropicMessagesResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse completion response: %w", err)
	}

	// Extract text content from response
	var content string
	for _, block := range resp.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}

	return &CompletionResponse{
		Content: content,
		Model:   resp.Model,
		Usage: &TokenUsage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
		FinishReason: resp.StopReason,
	}, nil
}

// Embed generates embeddings - Anthropic doesn't support embeddings natively.
func (p *AnthropicProvider) Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	return nil, fmt.Errorf("anthropic does not support embeddings")
}

// SuggestTags suggests tags for the given content.
func (p *AnthropicProvider) SuggestTags(ctx context.Context, req *SuggestTagsRequest) (*SuggestTagsResponse, error) {
	return p.DefaultSuggestTags(ctx, p, req)
}

// Summarize generates a summary of the given content.
func (p *AnthropicProvider) Summarize(ctx context.Context, req *SummarizeRequest) (*SummarizeResponse, error) {
	return p.DefaultSummarize(ctx, p, req)
}

// ToProto converts the provider configuration to proto format.
func (p *AnthropicProvider) ToProto() *storepb.LLMAnthropicConfig {
	return &storepb.LLMAnthropicConfig{
		ApiKey:       p.apiKey,
		BaseUrl:      p.baseURL,
		DefaultModel: p.defaultModel,
	}
}

// Anthropic API request/response types

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicMessagesRequest struct {
	Model       string             `json:"model"`
	Messages    []anthropicMessage `json:"messages"`
	System      string             `json:"system,omitempty"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature float64            `json:"temperature,omitempty"`
	TopP        float64            `json:"top_p,omitempty"`
}

type anthropicMessagesResponse struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Role         string `json:"role"`
	Model        string `json:"model"`
	StopReason   string `json:"stop_reason"`
	StopSequence string `json:"stop_sequence"`
	Content      []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}
