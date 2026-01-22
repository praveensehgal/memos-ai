package llm

import (
	"context"
	"testing"
)

// mockProvider is a mock implementation for testing.
type mockProvider struct {
	providerType  ProviderType
	name          string
	configured    bool
	defaultModel  string
	models        []string
	completeResp  *CompletionResponse
	completeErr   error
	embedResp     *EmbeddingResponse
	embedErr      error
	suggestResp   *SuggestTagsResponse
	suggestErr    error
	summarizeResp *SummarizeResponse
	summarizeErr  error
}

func (m *mockProvider) GetType() ProviderType {
	return m.providerType
}

func (m *mockProvider) GetName() string {
	return m.name
}

func (m *mockProvider) IsConfigured(ctx context.Context) bool {
	return m.configured
}

func (m *mockProvider) GetDefaultModel() string {
	return m.defaultModel
}

func (m *mockProvider) GetAvailableModels(ctx context.Context) ([]string, error) {
	return m.models, nil
}

func (m *mockProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	if m.completeErr != nil {
		return nil, m.completeErr
	}
	return m.completeResp, nil
}

func (m *mockProvider) Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	if m.embedErr != nil {
		return nil, m.embedErr
	}
	return m.embedResp, nil
}

func (m *mockProvider) SuggestTags(ctx context.Context, req *SuggestTagsRequest) (*SuggestTagsResponse, error) {
	if m.suggestErr != nil {
		return nil, m.suggestErr
	}
	return m.suggestResp, nil
}

func (m *mockProvider) Summarize(ctx context.Context, req *SummarizeRequest) (*SummarizeResponse, error) {
	if m.summarizeErr != nil {
		return nil, m.summarizeErr
	}
	return m.summarizeResp, nil
}

func TestProviderTypes(t *testing.T) {
	tests := []struct {
		providerType ProviderType
		expected     string
	}{
		{ProviderOpenAI, "openai"},
		{ProviderAnthropic, "anthropic"},
		{ProviderGemini, "gemini"},
		{ProviderOllama, "ollama"},
	}

	for _, tt := range tests {
		if string(tt.providerType) != tt.expected {
			t.Errorf("ProviderType %v: expected %s, got %s", tt.providerType, tt.expected, string(tt.providerType))
		}
	}
}

func TestRoleTypes(t *testing.T) {
	tests := []struct {
		role     Role
		expected string
	}{
		{RoleSystem, "system"},
		{RoleUser, "user"},
		{RoleAssistant, "assistant"},
	}

	for _, tt := range tests {
		if string(tt.role) != tt.expected {
			t.Errorf("Role %v: expected %s, got %s", tt.role, tt.expected, string(tt.role))
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	tests := []struct {
		providerType    ProviderType
		expectedModel   string
		expectedTimeout int
	}{
		{ProviderOpenAI, "gpt-4o-mini", 30},
		{ProviderAnthropic, "claude-3-5-sonnet-20241022", 30},
		{ProviderGemini, "gemini-1.5-flash", 30},
		{ProviderOllama, "llama3.2", 30},
	}

	for _, tt := range tests {
		config := DefaultConfig(tt.providerType)

		if config.Type != tt.providerType {
			t.Errorf("DefaultConfig(%v): expected type %v, got %v", tt.providerType, tt.providerType, config.Type)
		}

		if config.DefaultModel != tt.expectedModel {
			t.Errorf("DefaultConfig(%v): expected model %s, got %s", tt.providerType, tt.expectedModel, config.DefaultModel)
		}

		if config.Timeout != tt.expectedTimeout {
			t.Errorf("DefaultConfig(%v): expected timeout %d, got %d", tt.providerType, tt.expectedTimeout, config.Timeout)
		}
	}
}

func TestMessage(t *testing.T) {
	msg := Message{
		Role:    RoleUser,
		Content: "Hello, world!",
	}

	if msg.Role != RoleUser {
		t.Errorf("Message.Role: expected %v, got %v", RoleUser, msg.Role)
	}

	if msg.Content != "Hello, world!" {
		t.Errorf("Message.Content: expected 'Hello, world!', got '%s'", msg.Content)
	}
}

func TestCompletionRequest(t *testing.T) {
	req := CompletionRequest{
		Messages: []Message{
			{Role: RoleSystem, Content: "You are helpful."},
			{Role: RoleUser, Content: "Hello"},
		},
		Model:       "gpt-4",
		MaxTokens:   100,
		Temperature: 0.7,
	}

	if len(req.Messages) != 2 {
		t.Errorf("CompletionRequest.Messages: expected 2, got %d", len(req.Messages))
	}

	if req.Model != "gpt-4" {
		t.Errorf("CompletionRequest.Model: expected 'gpt-4', got '%s'", req.Model)
	}
}

func TestTokenUsage(t *testing.T) {
	usage := TokenUsage{
		PromptTokens:     10,
		CompletionTokens: 20,
		TotalTokens:      30,
	}

	if usage.TotalTokens != usage.PromptTokens+usage.CompletionTokens {
		t.Errorf("TokenUsage: total should equal prompt + completion")
	}
}
