// Package llm provides a unified interface for Large Language Model providers.
// It supports multiple providers (OpenAI, Anthropic, Gemini, Ollama) with a
// common interface for chat completion, embeddings, and AI-assisted features.
package llm

import (
	"context"
	"errors"
)

// Common errors for LLM operations.
var (
	// ErrProviderNotConfigured indicates the provider is not properly configured.
	ErrProviderNotConfigured = errors.New("llm provider not configured")

	// ErrInvalidAPIKey indicates the API key is invalid or missing.
	ErrInvalidAPIKey = errors.New("invalid or missing API key")

	// ErrRateLimited indicates the provider rate limit has been exceeded.
	ErrRateLimited = errors.New("rate limit exceeded")

	// ErrContextTooLong indicates the input exceeds the model's context window.
	ErrContextTooLong = errors.New("input exceeds maximum context length")

	// ErrModelNotFound indicates the requested model is not available.
	ErrModelNotFound = errors.New("model not found")

	// ErrProviderUnavailable indicates the provider service is unavailable.
	ErrProviderUnavailable = errors.New("provider service unavailable")
)

// ProviderType identifies the LLM provider.
type ProviderType string

const (
	// ProviderOpenAI is the OpenAI provider (GPT-4, GPT-3.5).
	ProviderOpenAI ProviderType = "openai"

	// ProviderAnthropic is the Anthropic provider (Claude).
	ProviderAnthropic ProviderType = "anthropic"

	// ProviderGemini is the Google AI provider (Gemini).
	ProviderGemini ProviderType = "gemini"

	// ProviderOllama is the local Ollama provider.
	ProviderOllama ProviderType = "ollama"
)

// Role represents the role of a message sender.
type Role string

const (
	// RoleSystem is for system prompts that set the AI's behavior.
	RoleSystem Role = "system"

	// RoleUser is for user messages.
	RoleUser Role = "user"

	// RoleAssistant is for AI assistant responses.
	RoleAssistant Role = "assistant"
)

// Message represents a single message in a conversation.
type Message struct {
	// Role indicates who sent the message (system, user, assistant).
	Role Role `json:"role"`

	// Content is the text content of the message.
	Content string `json:"content"`
}

// CompletionRequest contains parameters for a chat completion request.
type CompletionRequest struct {
	// Messages is the conversation history.
	Messages []Message `json:"messages"`

	// Model is the specific model to use (optional, uses provider default if empty).
	Model string `json:"model,omitempty"`

	// MaxTokens limits the response length.
	MaxTokens int `json:"max_tokens,omitempty"`

	// Temperature controls randomness (0.0-2.0, default varies by provider).
	Temperature float64 `json:"temperature,omitempty"`

	// TopP controls nucleus sampling (0.0-1.0).
	TopP float64 `json:"top_p,omitempty"`

	// Stream indicates whether to stream the response.
	Stream bool `json:"stream,omitempty"`
}

// CompletionResponse contains the result of a chat completion.
type CompletionResponse struct {
	// Content is the generated text response.
	Content string `json:"content"`

	// Model is the actual model used.
	Model string `json:"model"`

	// Usage contains token usage statistics.
	Usage *TokenUsage `json:"usage,omitempty"`

	// FinishReason indicates why the generation stopped.
	FinishReason string `json:"finish_reason,omitempty"`
}

// TokenUsage tracks token consumption for billing/monitoring.
type TokenUsage struct {
	// PromptTokens is the number of tokens in the prompt.
	PromptTokens int `json:"prompt_tokens"`

	// CompletionTokens is the number of tokens in the completion.
	CompletionTokens int `json:"completion_tokens"`

	// TotalTokens is the sum of prompt and completion tokens.
	TotalTokens int `json:"total_tokens"`
}

// EmbeddingRequest contains parameters for an embedding request.
type EmbeddingRequest struct {
	// Input is the text to embed (can be a single string or array).
	Input []string `json:"input"`

	// Model is the specific embedding model to use (optional).
	Model string `json:"model,omitempty"`

	// Dimensions specifies the output embedding dimensions (if supported).
	Dimensions int `json:"dimensions,omitempty"`
}

// EmbeddingResponse contains the result of an embedding request.
type EmbeddingResponse struct {
	// Embeddings is the generated vector embeddings (one per input).
	Embeddings [][]float32 `json:"embeddings"`

	// Model is the actual model used.
	Model string `json:"model"`

	// Usage contains token usage statistics.
	Usage *TokenUsage `json:"usage,omitempty"`
}

// SuggestTagsRequest contains parameters for tag suggestion.
type SuggestTagsRequest struct {
	// Content is the memo content to analyze.
	Content string `json:"content"`

	// ExistingTags are tags already in the system (for consistency).
	ExistingTags []string `json:"existing_tags,omitempty"`

	// MaxTags is the maximum number of tags to suggest.
	MaxTags int `json:"max_tags,omitempty"`

	// Language is the preferred language for tags (e.g., "en", "zh").
	Language string `json:"language,omitempty"`
}

// SuggestTagsResponse contains suggested tags for content.
type SuggestTagsResponse struct {
	// Tags is the list of suggested tags.
	Tags []string `json:"tags"`

	// Confidence scores for each tag (0.0-1.0).
	Confidence []float64 `json:"confidence,omitempty"`
}

// SummarizeRequest contains parameters for content summarization.
type SummarizeRequest struct {
	// Content is the text to summarize.
	Content string `json:"content"`

	// MaxLength is the maximum summary length in characters.
	MaxLength int `json:"max_length,omitempty"`

	// Style is the summarization style (e.g., "brief", "detailed", "bullet").
	Style string `json:"style,omitempty"`
}

// SummarizeResponse contains the summarized content.
type SummarizeResponse struct {
	// Summary is the generated summary.
	Summary string `json:"summary"`

	// KeyPoints are the main points extracted (optional).
	KeyPoints []string `json:"key_points,omitempty"`
}

// Provider defines the interface for LLM providers.
// All providers must implement these methods to be used with Memos AI.
type Provider interface {
	// GetType returns the provider type identifier.
	GetType() ProviderType

	// GetName returns a human-readable name for the provider.
	GetName() string

	// IsConfigured checks if the provider has valid configuration.
	IsConfigured(ctx context.Context) bool

	// GetDefaultModel returns the default model for this provider.
	GetDefaultModel() string

	// GetAvailableModels returns a list of available models.
	GetAvailableModels(ctx context.Context) ([]string, error)

	// Complete performs a chat completion request.
	Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)

	// Embed generates vector embeddings for the given input.
	Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error)

	// SuggestTags suggests relevant tags for content.
	SuggestTags(ctx context.Context, req *SuggestTagsRequest) (*SuggestTagsResponse, error)

	// Summarize generates a summary of the content.
	Summarize(ctx context.Context, req *SummarizeRequest) (*SummarizeResponse, error)
}

// ProviderConfig holds configuration for creating a provider.
type ProviderConfig struct {
	// Type is the provider type.
	Type ProviderType `json:"type"`

	// APIKey is the authentication key (not needed for Ollama).
	APIKey string `json:"api_key,omitempty"`

	// BaseURL overrides the default API endpoint.
	BaseURL string `json:"base_url,omitempty"`

	// DefaultModel is the model to use when not specified.
	DefaultModel string `json:"default_model,omitempty"`

	// EmbeddingModel is the model to use for embeddings.
	EmbeddingModel string `json:"embedding_model,omitempty"`

	// OllamaHost is the Ollama server address (only for Ollama provider).
	OllamaHost string `json:"ollama_host,omitempty"`

	// Timeout is the request timeout in seconds.
	Timeout int `json:"timeout,omitempty"`

	// MaxRetries is the number of retries for failed requests.
	MaxRetries int `json:"max_retries,omitempty"`
}

// DefaultConfig returns sensible defaults for the given provider type.
func DefaultConfig(providerType ProviderType) *ProviderConfig {
	config := &ProviderConfig{
		Type:       providerType,
		Timeout:    30,
		MaxRetries: 3,
	}

	switch providerType {
	case ProviderOpenAI:
		config.BaseURL = "https://api.openai.com/v1"
		config.DefaultModel = "gpt-4o-mini"
	case ProviderAnthropic:
		config.BaseURL = "https://api.anthropic.com"
		config.DefaultModel = "claude-3-5-sonnet-20241022"
	case ProviderGemini:
		config.BaseURL = "https://generativelanguage.googleapis.com/v1beta"
		config.DefaultModel = "gemini-1.5-flash"
	case ProviderOllama:
		config.OllamaHost = "http://localhost:11434"
		config.DefaultModel = "llama3.2"
	}

	return config
}
