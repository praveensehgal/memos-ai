package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	storepb "github.com/usememos/memos/proto/gen/store"
)

func TestNewOpenAIProvider(t *testing.T) {
	config := &ProviderConfig{
		Type:   ProviderOpenAI,
		APIKey: "test-key",
	}

	provider := NewOpenAIProvider(config)

	if provider.GetType() != ProviderOpenAI {
		t.Errorf("Expected type %v, got %v", ProviderOpenAI, provider.GetType())
	}

	if provider.GetName() != "OpenAI" {
		t.Errorf("Expected name 'OpenAI', got '%s'", provider.GetName())
	}

	if provider.GetDefaultModel() != openAIDefaultModel {
		t.Errorf("Expected default model '%s', got '%s'", openAIDefaultModel, provider.GetDefaultModel())
	}
}

func TestNewOpenAIProviderCustomConfig(t *testing.T) {
	config := &ProviderConfig{
		Type:           ProviderOpenAI,
		APIKey:         "test-key",
		BaseURL:        "https://custom.openai.com/v1",
		DefaultModel:   "gpt-4",
		EmbeddingModel: "text-embedding-ada-002",
	}

	provider := NewOpenAIProvider(config)

	if provider.baseURL != "https://custom.openai.com/v1" {
		t.Errorf("Expected custom base URL, got '%s'", provider.baseURL)
	}

	if provider.GetDefaultModel() != "gpt-4" {
		t.Errorf("Expected custom default model 'gpt-4', got '%s'", provider.GetDefaultModel())
	}

	if provider.embeddingModel != "text-embedding-ada-002" {
		t.Errorf("Expected custom embedding model, got '%s'", provider.embeddingModel)
	}
}

func TestNewOpenAIProviderFromProto(t *testing.T) {
	pbConfig := &storepb.LLMOpenAIConfig{
		ApiKey:         "proto-test-key",
		BaseUrl:        "https://azure.openai.com/v1",
		DefaultModel:   "gpt-4-turbo",
		EmbeddingModel: "text-embedding-3-large",
	}

	provider := NewOpenAIProviderFromProto(pbConfig)

	if provider.apiKey != "proto-test-key" {
		t.Errorf("Expected API key from proto config")
	}

	if provider.baseURL != "https://azure.openai.com/v1" {
		t.Errorf("Expected base URL from proto config")
	}

	if provider.GetDefaultModel() != "gpt-4-turbo" {
		t.Errorf("Expected default model from proto config")
	}
}

func TestOpenAIProviderIsConfigured(t *testing.T) {
	ctx := context.Background()

	// Without API key
	provider := NewOpenAIProvider(&ProviderConfig{Type: ProviderOpenAI})
	if provider.IsConfigured(ctx) {
		t.Error("Expected not configured without API key")
	}

	// With API key
	provider = NewOpenAIProvider(&ProviderConfig{
		Type:   ProviderOpenAI,
		APIKey: "test-key",
	})
	if !provider.IsConfigured(ctx) {
		t.Error("Expected configured with API key")
	}
}

func TestOpenAIProviderComplete(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("Expected path /chat/completions, got %s", r.URL.Path)
		}

		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		// Check authorization header
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-key" {
			t.Errorf("Expected Bearer token, got %s", auth)
		}

		// Parse request body
		var req openAIChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		if req.Model != "gpt-4o-mini" {
			t.Errorf("Expected model gpt-4o-mini, got %s", req.Model)
		}

		if len(req.Messages) != 1 {
			t.Errorf("Expected 1 message, got %d", len(req.Messages))
		}

		// Return mock response
		resp := openAIChatResponse{
			ID:      "chatcmpl-123",
			Object:  "chat.completion",
			Created: 1677652288,
			Model:   "gpt-4o-mini",
			Choices: []struct {
				Index   int `json:"index"`
				Message struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			}{
				{
					Index: 0,
					Message: struct {
						Role    string `json:"role"`
						Content string `json:"content"`
					}{
						Role:    "assistant",
						Content: "Hello! How can I help you today?",
					},
					FinishReason: "stop",
				},
			},
			Usage: struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			}{
				PromptTokens:     5,
				CompletionTokens: 10,
				TotalTokens:      15,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := NewOpenAIProvider(&ProviderConfig{
		Type:    ProviderOpenAI,
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	req := &CompletionRequest{
		Messages: []Message{
			{Role: RoleUser, Content: "Hello"},
		},
	}

	resp, err := provider.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete() error: %v", err)
	}

	if resp.Content != "Hello! How can I help you today?" {
		t.Errorf("Expected content 'Hello! How can I help you today?', got '%s'", resp.Content)
	}

	if resp.Model != "gpt-4o-mini" {
		t.Errorf("Expected model 'gpt-4o-mini', got '%s'", resp.Model)
	}

	if resp.Usage.TotalTokens != 15 {
		t.Errorf("Expected 15 total tokens, got %d", resp.Usage.TotalTokens)
	}
}

func TestOpenAIProviderCompleteNotConfigured(t *testing.T) {
	provider := NewOpenAIProvider(&ProviderConfig{Type: ProviderOpenAI})

	req := &CompletionRequest{
		Messages: []Message{
			{Role: RoleUser, Content: "Hello"},
		},
	}

	_, err := provider.Complete(context.Background(), req)
	if err != ErrProviderNotConfigured {
		t.Errorf("Expected ErrProviderNotConfigured, got %v", err)
	}
}

func TestOpenAIProviderEmbed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/embeddings" {
			t.Errorf("Expected path /embeddings, got %s", r.URL.Path)
		}

		var req openAIEmbeddingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		if req.Model != openAIEmbeddingModel {
			t.Errorf("Expected model %s, got %s", openAIEmbeddingModel, req.Model)
		}

		resp := openAIEmbeddingResponse{
			Object: "list",
			Data: []struct {
				Object    string    `json:"object"`
				Index     int       `json:"index"`
				Embedding []float32 `json:"embedding"`
			}{
				{
					Object:    "embedding",
					Index:     0,
					Embedding: []float32{0.1, 0.2, 0.3, 0.4, 0.5},
				},
			},
			Model: openAIEmbeddingModel,
			Usage: struct {
				PromptTokens int `json:"prompt_tokens"`
				TotalTokens  int `json:"total_tokens"`
			}{
				PromptTokens: 3,
				TotalTokens:  3,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := NewOpenAIProvider(&ProviderConfig{
		Type:    ProviderOpenAI,
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	req := &EmbeddingRequest{
		Input: []string{"Hello world"},
	}

	resp, err := provider.Embed(context.Background(), req)
	if err != nil {
		t.Fatalf("Embed() error: %v", err)
	}

	if len(resp.Embeddings) != 1 {
		t.Fatalf("Expected 1 embedding, got %d", len(resp.Embeddings))
	}

	if len(resp.Embeddings[0]) != 5 {
		t.Errorf("Expected embedding of length 5, got %d", len(resp.Embeddings[0]))
	}

	if resp.Model != openAIEmbeddingModel {
		t.Errorf("Expected model %s, got %s", openAIEmbeddingModel, resp.Model)
	}
}

func TestOpenAIProviderEmbedNotConfigured(t *testing.T) {
	provider := NewOpenAIProvider(&ProviderConfig{Type: ProviderOpenAI})

	req := &EmbeddingRequest{
		Input: []string{"Hello world"},
	}

	_, err := provider.Embed(context.Background(), req)
	if err != ErrProviderNotConfigured {
		t.Errorf("Expected ErrProviderNotConfigured, got %v", err)
	}
}

func TestOpenAIProviderGetAvailableModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			t.Errorf("Expected path /models, got %s", r.URL.Path)
		}

		if r.Method != http.MethodGet {
			t.Errorf("Expected GET method, got %s", r.Method)
		}

		resp := openAIModelsResponse{
			Object: "list",
			Data: []struct {
				ID      string `json:"id"`
				Object  string `json:"object"`
				Created int64  `json:"created"`
				OwnedBy string `json:"owned_by"`
			}{
				{ID: "gpt-4o-mini", Object: "model", Created: 1677610602, OwnedBy: "openai"},
				{ID: "gpt-4", Object: "model", Created: 1677610602, OwnedBy: "openai"},
				{ID: "gpt-3.5-turbo", Object: "model", Created: 1677610602, OwnedBy: "openai"},
				{ID: "text-embedding-3-small", Object: "model", Created: 1677610602, OwnedBy: "openai"},
				{ID: "whisper-1", Object: "model", Created: 1677610602, OwnedBy: "openai"},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := NewOpenAIProvider(&ProviderConfig{
		Type:    ProviderOpenAI,
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	models, err := provider.GetAvailableModels(context.Background())
	if err != nil {
		t.Fatalf("GetAvailableModels() error: %v", err)
	}

	// Should only include chat models (gpt-*, o1-*)
	expectedCount := 3 // gpt-4o-mini, gpt-4, gpt-3.5-turbo
	if len(models) != expectedCount {
		t.Errorf("Expected %d chat models, got %d: %v", expectedCount, len(models), models)
	}

	// Verify specific models are included
	hasGPT4 := false
	for _, m := range models {
		if m == "gpt-4" {
			hasGPT4 = true
			break
		}
	}
	if !hasGPT4 {
		t.Error("Expected gpt-4 in models list")
	}
}

func TestIsOpenAIChatModel(t *testing.T) {
	tests := []struct {
		id       string
		expected bool
	}{
		{"gpt-4", true},
		{"gpt-4o-mini", true},
		{"gpt-4-turbo", true},
		{"gpt-3.5-turbo", true},
		{"gpt-3.5-turbo-16k", true},
		{"o1-preview", true},
		{"o1-mini", true},
		{"chatgpt-4o-latest", true},
		{"text-embedding-3-small", false},
		{"whisper-1", false},
		{"dall-e-3", false},
		{"tts-1", false},
	}

	for _, tt := range tests {
		result := isOpenAIChatModel(tt.id)
		if result != tt.expected {
			t.Errorf("isOpenAIChatModel(%q): expected %v, got %v", tt.id, tt.expected, result)
		}
	}
}

func TestOpenAIProviderSuggestTags(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("Expected path /chat/completions, got %s", r.URL.Path)
		}

		resp := openAIChatResponse{
			ID:    "chatcmpl-123",
			Model: "gpt-4o-mini",
			Choices: []struct {
				Index   int `json:"index"`
				Message struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			}{
				{
					Index: 0,
					Message: struct {
						Role    string `json:"role"`
						Content string `json:"content"`
					}{
						Role:    "assistant",
						Content: `["meeting", "project", "notes"]`,
					},
					FinishReason: "stop",
				},
			},
			Usage: struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			}{
				PromptTokens:     50,
				CompletionTokens: 10,
				TotalTokens:      60,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := NewOpenAIProvider(&ProviderConfig{
		Type:    ProviderOpenAI,
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	req := &SuggestTagsRequest{
		Content: "Meeting notes for project Alpha",
		MaxTags: 5,
	}

	resp, err := provider.SuggestTags(context.Background(), req)
	if err != nil {
		t.Fatalf("SuggestTags() error: %v", err)
	}

	if len(resp.Tags) != 3 {
		t.Errorf("Expected 3 tags, got %d: %v", len(resp.Tags), resp.Tags)
	}
}

func TestOpenAIProviderSummarize(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("Expected path /chat/completions, got %s", r.URL.Path)
		}

		resp := openAIChatResponse{
			ID:    "chatcmpl-123",
			Model: "gpt-4o-mini",
			Choices: []struct {
				Index   int `json:"index"`
				Message struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			}{
				{
					Index: 0,
					Message: struct {
						Role    string `json:"role"`
						Content string `json:"content"`
					}{
						Role:    "assistant",
						Content: "This is a brief summary of the meeting notes.",
					},
					FinishReason: "stop",
				},
			},
			Usage: struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			}{
				PromptTokens:     100,
				CompletionTokens: 20,
				TotalTokens:      120,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := NewOpenAIProvider(&ProviderConfig{
		Type:    ProviderOpenAI,
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	req := &SummarizeRequest{
		Content:   "Long meeting notes that need to be summarized...",
		MaxLength: 100,
	}

	resp, err := provider.Summarize(context.Background(), req)
	if err != nil {
		t.Fatalf("Summarize() error: %v", err)
	}

	if resp.Summary != "This is a brief summary of the meeting notes." {
		t.Errorf("Unexpected summary: %s", resp.Summary)
	}
}

func TestOpenAIProviderHTTPErrors(t *testing.T) {
	tests := []struct {
		statusCode    int
		expectedError error
	}{
		{401, ErrInvalidAPIKey},
		{429, ErrRateLimited},
		{503, ErrProviderUnavailable},
	}

	for _, tt := range tests {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(tt.statusCode)
			w.Write([]byte(`{"error": {"message": "test error"}}`))
		}))

		provider := NewOpenAIProvider(&ProviderConfig{
			Type:       ProviderOpenAI,
			APIKey:     "test-key",
			BaseURL:    server.URL,
			MaxRetries: 0, // Disable retries for test
		})

		req := &CompletionRequest{
			Messages: []Message{{Role: RoleUser, Content: "Hello"}},
		}

		_, err := provider.Complete(context.Background(), req)
		if err != tt.expectedError {
			t.Errorf("Status %d: expected %v, got %v", tt.statusCode, tt.expectedError, err)
		}

		server.Close()
	}
}
