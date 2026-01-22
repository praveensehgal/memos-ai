package llm

import (
	"context"
	"testing"
)

func TestNewService(t *testing.T) {
	svc := NewService()
	if svc == nil {
		t.Fatal("NewService() returned nil")
	}

	// Should have no providers initially
	providers := svc.ListProviders()
	if len(providers) != 0 {
		t.Errorf("Expected 0 providers, got %d", len(providers))
	}

	// Should not be configured
	if svc.IsConfigured(context.Background()) {
		t.Error("Expected service to not be configured initially")
	}
}

func TestRegisterProvider(t *testing.T) {
	svc := NewService()

	// Create a mock provider
	provider := &mockProvider{
		providerType: ProviderOpenAI,
		name:         "OpenAI",
		configured:   true,
		defaultModel: "gpt-4",
	}

	// Register the provider
	err := svc.RegisterProvider(provider)
	if err != nil {
		t.Fatalf("RegisterProvider() error: %v", err)
	}

	// Should have one provider
	providers := svc.ListProviders()
	if len(providers) != 1 {
		t.Errorf("Expected 1 provider, got %d", len(providers))
	}

	// Should be configured now
	if !svc.IsConfigured(context.Background()) {
		t.Error("Expected service to be configured after adding configured provider")
	}

	// Provider should be auto-selected as active
	activeProvider := svc.GetProvider()
	if activeProvider == nil {
		t.Fatal("Expected active provider, got nil")
	}
	if activeProvider.GetType() != ProviderOpenAI {
		t.Errorf("Expected active provider to be OpenAI, got %v", activeProvider.GetType())
	}
}

func TestRegisterNilProvider(t *testing.T) {
	svc := NewService()

	err := svc.RegisterProvider(nil)
	if err == nil {
		t.Error("Expected error when registering nil provider")
	}
}

func TestSetActiveProvider(t *testing.T) {
	svc := NewService()

	// Register two providers
	openai := &mockProvider{
		providerType: ProviderOpenAI,
		name:         "OpenAI",
		configured:   true,
	}
	ollama := &mockProvider{
		providerType: ProviderOllama,
		name:         "Ollama",
		configured:   true,
	}

	svc.RegisterProvider(openai)
	svc.RegisterProvider(ollama)

	// OpenAI should be auto-selected (first configured)
	if svc.GetProvider().GetType() != ProviderOpenAI {
		t.Error("Expected OpenAI to be auto-selected")
	}

	// Switch to Ollama
	err := svc.SetActiveProvider(ProviderOllama)
	if err != nil {
		t.Fatalf("SetActiveProvider() error: %v", err)
	}

	if svc.GetProvider().GetType() != ProviderOllama {
		t.Error("Expected Ollama to be active after switch")
	}
}

func TestSetActiveProviderNotRegistered(t *testing.T) {
	svc := NewService()

	err := svc.SetActiveProvider(ProviderOpenAI)
	if err == nil {
		t.Error("Expected error when setting unregistered provider as active")
	}
}

func TestGetProviderByType(t *testing.T) {
	svc := NewService()

	provider := &mockProvider{
		providerType: ProviderOpenAI,
		name:         "OpenAI",
		configured:   true,
	}
	svc.RegisterProvider(provider)

	// Get existing provider
	p, err := svc.GetProviderByType(ProviderOpenAI)
	if err != nil {
		t.Fatalf("GetProviderByType() error: %v", err)
	}
	if p.GetType() != ProviderOpenAI {
		t.Errorf("Expected OpenAI provider, got %v", p.GetType())
	}

	// Get non-existing provider
	_, err = svc.GetProviderByType(ProviderAnthropic)
	if err == nil {
		t.Error("Expected error for non-existing provider")
	}
}

func TestListProviders(t *testing.T) {
	svc := NewService()

	// Register multiple providers
	openai := &mockProvider{
		providerType: ProviderOpenAI,
		name:         "OpenAI",
		configured:   true,
		defaultModel: "gpt-4",
	}
	ollama := &mockProvider{
		providerType: ProviderOllama,
		name:         "Ollama",
		configured:   false,
		defaultModel: "llama3.2",
	}

	svc.RegisterProvider(openai)
	svc.RegisterProvider(ollama)

	providers := svc.ListProviders()
	if len(providers) != 2 {
		t.Fatalf("Expected 2 providers, got %d", len(providers))
	}

	// Find and verify each provider
	var foundOpenAI, foundOllama bool
	for _, p := range providers {
		switch p.Type {
		case ProviderOpenAI:
			foundOpenAI = true
			if !p.Configured {
				t.Error("OpenAI should be configured")
			}
			if !p.Active {
				t.Error("OpenAI should be active (auto-selected)")
			}
		case ProviderOllama:
			foundOllama = true
			if p.Configured {
				t.Error("Ollama should not be configured")
			}
			if p.Active {
				t.Error("Ollama should not be active")
			}
		}
	}

	if !foundOpenAI {
		t.Error("OpenAI provider not found in list")
	}
	if !foundOllama {
		t.Error("Ollama provider not found in list")
	}
}

func TestServiceComplete(t *testing.T) {
	svc := NewService()

	expectedResp := &CompletionResponse{
		Content: "Hello! How can I help you?",
		Model:   "gpt-4",
		Usage: &TokenUsage{
			PromptTokens:     5,
			CompletionTokens: 10,
			TotalTokens:      15,
		},
	}

	provider := &mockProvider{
		providerType: ProviderOpenAI,
		name:         "OpenAI",
		configured:   true,
		completeResp: expectedResp,
	}
	svc.RegisterProvider(provider)

	req := &CompletionRequest{
		Messages: []Message{
			{Role: RoleUser, Content: "Hello"},
		},
	}

	resp, err := svc.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete() error: %v", err)
	}

	if resp.Content != expectedResp.Content {
		t.Errorf("Expected content '%s', got '%s'", expectedResp.Content, resp.Content)
	}
}

func TestServiceCompleteNoProvider(t *testing.T) {
	svc := NewService()

	req := &CompletionRequest{
		Messages: []Message{
			{Role: RoleUser, Content: "Hello"},
		},
	}

	_, err := svc.Complete(context.Background(), req)
	if err != ErrProviderNotConfigured {
		t.Errorf("Expected ErrProviderNotConfigured, got %v", err)
	}
}

func TestServiceEmbed(t *testing.T) {
	svc := NewService()

	expectedResp := &EmbeddingResponse{
		Embeddings: [][]float32{{0.1, 0.2, 0.3}},
		Model:      "text-embedding-3-small",
	}

	provider := &mockProvider{
		providerType: ProviderOpenAI,
		name:         "OpenAI",
		configured:   true,
		embedResp:    expectedResp,
	}
	svc.RegisterProvider(provider)

	req := &EmbeddingRequest{
		Input: []string{"Hello world"},
	}

	resp, err := svc.Embed(context.Background(), req)
	if err != nil {
		t.Fatalf("Embed() error: %v", err)
	}

	if len(resp.Embeddings) != 1 {
		t.Errorf("Expected 1 embedding, got %d", len(resp.Embeddings))
	}
}

func TestServiceSuggestTags(t *testing.T) {
	svc := NewService()

	expectedResp := &SuggestTagsResponse{
		Tags: []string{"meeting", "project", "todo"},
	}

	provider := &mockProvider{
		providerType: ProviderOpenAI,
		name:         "OpenAI",
		configured:   true,
		suggestResp:  expectedResp,
	}
	svc.RegisterProvider(provider)

	req := &SuggestTagsRequest{
		Content: "Meeting notes for project Alpha",
		MaxTags: 5,
	}

	resp, err := svc.SuggestTags(context.Background(), req)
	if err != nil {
		t.Fatalf("SuggestTags() error: %v", err)
	}

	if len(resp.Tags) != 3 {
		t.Errorf("Expected 3 tags, got %d", len(resp.Tags))
	}
}

func TestServiceSummarize(t *testing.T) {
	svc := NewService()

	expectedResp := &SummarizeResponse{
		Summary: "This is a brief summary.",
	}

	provider := &mockProvider{
		providerType:  ProviderOpenAI,
		name:          "OpenAI",
		configured:    true,
		summarizeResp: expectedResp,
	}
	svc.RegisterProvider(provider)

	req := &SummarizeRequest{
		Content:   "A very long content that needs summarization...",
		MaxLength: 100,
	}

	resp, err := svc.Summarize(context.Background(), req)
	if err != nil {
		t.Fatalf("Summarize() error: %v", err)
	}

	if resp.Summary != expectedResp.Summary {
		t.Errorf("Expected summary '%s', got '%s'", expectedResp.Summary, resp.Summary)
	}
}
