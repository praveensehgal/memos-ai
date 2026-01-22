package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	storepb "github.com/usememos/memos/proto/gen/store"
)

func TestNewOllamaProvider(t *testing.T) {
	config := &ProviderConfig{
		Type: ProviderOllama,
	}

	provider := NewOllamaProvider(config)

	if provider.GetType() != ProviderOllama {
		t.Errorf("Expected type %v, got %v", ProviderOllama, provider.GetType())
	}

	if provider.GetName() != "Ollama" {
		t.Errorf("Expected name 'Ollama', got %s", provider.GetName())
	}

	// Should use defaults
	if provider.host != ollamaDefaultHost {
		t.Errorf("Expected default host %s, got %s", ollamaDefaultHost, provider.host)
	}

	if provider.defaultModel != ollamaDefaultModel {
		t.Errorf("Expected default model %s, got %s", ollamaDefaultModel, provider.defaultModel)
	}

	if provider.embeddingModel != ollamaDefaultEmbeddingModel {
		t.Errorf("Expected default embedding model %s, got %s", ollamaDefaultEmbeddingModel, provider.embeddingModel)
	}
}

func TestNewOllamaProviderCustomConfig(t *testing.T) {
	config := &ProviderConfig{
		Type:           ProviderOllama,
		OllamaHost:     "http://192.168.1.100:11434",
		DefaultModel:   "mistral",
		EmbeddingModel: "mxbai-embed-large",
	}

	provider := NewOllamaProvider(config)

	if provider.host != "http://192.168.1.100:11434" {
		t.Errorf("Expected custom host, got %s", provider.host)
	}

	if provider.defaultModel != "mistral" {
		t.Errorf("Expected custom model 'mistral', got %s", provider.defaultModel)
	}

	if provider.embeddingModel != "mxbai-embed-large" {
		t.Errorf("Expected custom embedding model, got %s", provider.embeddingModel)
	}
}

func TestNewOllamaProviderFromProto(t *testing.T) {
	pbConfig := &storepb.LLMOllamaConfig{
		Host:           "http://ollama.local:11434",
		DefaultModel:   "codellama",
		EmbeddingModel: "nomic-embed-text",
	}

	provider := NewOllamaProviderFromProto(pbConfig)

	if provider.host != "http://ollama.local:11434" {
		t.Errorf("Expected host from proto, got %s", provider.host)
	}

	if provider.defaultModel != "codellama" {
		t.Errorf("Expected model from proto, got %s", provider.defaultModel)
	}

	if provider.embeddingModel != "nomic-embed-text" {
		t.Errorf("Expected embedding model from proto, got %s", provider.embeddingModel)
	}
}

func TestOllamaProviderIsConfigured(t *testing.T) {
	ctx := context.Background()

	// With host set (default)
	provider := NewOllamaProvider(&ProviderConfig{
		Type: ProviderOllama,
	})
	if !provider.IsConfigured(ctx) {
		t.Error("Expected provider to be configured with default host")
	}

	// With empty host
	provider.host = ""
	if provider.IsConfigured(ctx) {
		t.Error("Expected provider to not be configured with empty host")
	}
}

func TestOllamaProviderComplete(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("Expected path /api/chat, got %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		// Verify request body
		var req ollamaChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		if req.Model != "llama3.2" {
			t.Errorf("Expected model 'llama3.2', got %s", req.Model)
		}
		if len(req.Messages) != 1 {
			t.Errorf("Expected 1 message, got %d", len(req.Messages))
		}

		resp := ollamaChatResponse{
			Model: "llama3.2",
			Message: struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			}{
				Role:    "assistant",
				Content: "Hello! How can I assist you today?",
			},
			Done:            true,
			DoneReason:      "stop",
			PromptEvalCount: 10,
			EvalCount:       15,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := NewOllamaProvider(&ProviderConfig{
		Type:       ProviderOllama,
		OllamaHost: server.URL,
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

	if resp.Content != "Hello! How can I assist you today?" {
		t.Errorf("Unexpected content: %s", resp.Content)
	}

	if resp.Model != "llama3.2" {
		t.Errorf("Expected model 'llama3.2', got %s", resp.Model)
	}

	if resp.Usage == nil {
		t.Fatal("Expected usage info")
	}

	if resp.Usage.PromptTokens != 10 {
		t.Errorf("Expected 10 prompt tokens, got %d", resp.Usage.PromptTokens)
	}

	if resp.Usage.CompletionTokens != 15 {
		t.Errorf("Expected 15 completion tokens, got %d", resp.Usage.CompletionTokens)
	}
}

func TestOllamaProviderCompleteNotConfigured(t *testing.T) {
	provider := NewOllamaProvider(&ProviderConfig{
		Type: ProviderOllama,
	})
	provider.host = "" // Unconfigure

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

func TestOllamaProviderCompleteWithOptions(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ollamaChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		// Verify options are passed
		if req.Options == nil {
			t.Error("Expected options to be set")
		} else {
			if req.Options.Temperature != 0.7 {
				t.Errorf("Expected temperature 0.7, got %f", req.Options.Temperature)
			}
			if req.Options.TopP != 0.9 {
				t.Errorf("Expected top_p 0.9, got %f", req.Options.TopP)
			}
		}

		resp := ollamaChatResponse{
			Model: "llama3.2",
			Message: struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			}{
				Role:    "assistant",
				Content: "Response",
			},
			Done: true,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := NewOllamaProvider(&ProviderConfig{
		Type:       ProviderOllama,
		OllamaHost: server.URL,
	})

	req := &CompletionRequest{
		Messages: []Message{
			{Role: RoleUser, Content: "Hello"},
		},
		Temperature: 0.7,
		TopP:        0.9,
	}

	_, err := provider.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete() error: %v", err)
	}
}

func TestOllamaProviderEmbed(t *testing.T) {
	callCount := 0

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embed" {
			t.Errorf("Expected path /api/embed, got %s", r.URL.Path)
		}

		var req ollamaEmbedRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		callCount++

		resp := ollamaEmbedResponse{
			Model:           "nomic-embed-text",
			Embeddings:      [][]float32{{0.1, 0.2, 0.3}},
			PromptEvalCount: 5,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := NewOllamaProvider(&ProviderConfig{
		Type:       ProviderOllama,
		OllamaHost: server.URL,
	})

	req := &EmbeddingRequest{
		Input: []string{"Hello world", "Another text"},
	}

	resp, err := provider.Embed(context.Background(), req)
	if err != nil {
		t.Fatalf("Embed() error: %v", err)
	}

	// Should make 2 requests (one per input)
	if callCount != 2 {
		t.Errorf("Expected 2 API calls, got %d", callCount)
	}

	if len(resp.Embeddings) != 2 {
		t.Fatalf("Expected 2 embeddings, got %d", len(resp.Embeddings))
	}

	if resp.Model != "nomic-embed-text" {
		t.Errorf("Expected model 'nomic-embed-text', got %s", resp.Model)
	}

	if resp.Usage.TotalTokens != 10 {
		t.Errorf("Expected 10 total tokens, got %d", resp.Usage.TotalTokens)
	}
}

func TestOllamaProviderEmbedNotConfigured(t *testing.T) {
	provider := NewOllamaProvider(&ProviderConfig{
		Type: ProviderOllama,
	})
	provider.host = "" // Unconfigure

	req := &EmbeddingRequest{
		Input: []string{"Hello world"},
	}

	_, err := provider.Embed(context.Background(), req)
	if err != ErrProviderNotConfigured {
		t.Errorf("Expected ErrProviderNotConfigured, got %v", err)
	}
}

func TestOllamaProviderGetAvailableModels(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Errorf("Expected path /api/tags, got %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET method, got %s", r.Method)
		}

		resp := ollamaModelsResponse{
			Models: []struct {
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
			}{
				{Name: "llama3.2:latest", Model: "llama3.2:latest"},
				{Name: "mistral:latest", Model: "mistral:latest"},
				{Name: "nomic-embed-text:latest", Model: "nomic-embed-text:latest"},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := NewOllamaProvider(&ProviderConfig{
		Type:       ProviderOllama,
		OllamaHost: server.URL,
	})

	models, err := provider.GetAvailableModels(context.Background())
	if err != nil {
		t.Fatalf("GetAvailableModels() error: %v", err)
	}

	if len(models) != 3 {
		t.Fatalf("Expected 3 models, got %d", len(models))
	}

	expectedModels := []string{"llama3.2:latest", "mistral:latest", "nomic-embed-text:latest"}
	for i, expected := range expectedModels {
		if models[i] != expected {
			t.Errorf("Expected model %s at index %d, got %s", expected, i, models[i])
		}
	}
}

func TestOllamaProviderCheckHealth(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/version" {
			t.Errorf("Expected path /api/version, got %s", r.URL.Path)
		}

		resp := map[string]string{"version": "0.1.0"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := NewOllamaProvider(&ProviderConfig{
		Type:       ProviderOllama,
		OllamaHost: server.URL,
	})

	err := provider.CheckHealth(context.Background())
	if err != nil {
		t.Errorf("CheckHealth() error: %v", err)
	}
}

func TestOllamaProviderCheckHealthNotConfigured(t *testing.T) {
	provider := NewOllamaProvider(&ProviderConfig{
		Type: ProviderOllama,
	})
	provider.host = "" // Unconfigure

	err := provider.CheckHealth(context.Background())
	if err != ErrProviderNotConfigured {
		t.Errorf("Expected ErrProviderNotConfigured, got %v", err)
	}
}

func TestOllamaProviderSuggestTags(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ollamaChatResponse{
			Model: "llama3.2",
			Message: struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			}{
				Role:    "assistant",
				Content: "meeting, project, notes",
			},
			Done: true,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := NewOllamaProvider(&ProviderConfig{
		Type:       ProviderOllama,
		OllamaHost: server.URL,
	})

	req := &SuggestTagsRequest{
		Content: "Meeting notes for project Alpha",
		MaxTags: 5,
	}

	resp, err := provider.SuggestTags(context.Background(), req)
	if err != nil {
		t.Fatalf("SuggestTags() error: %v", err)
	}

	if len(resp.Tags) == 0 {
		t.Error("Expected some tags")
	}
}

func TestOllamaProviderSummarize(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ollamaChatResponse{
			Model: "llama3.2",
			Message: struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			}{
				Role:    "assistant",
				Content: "This is a concise summary of the content.",
			},
			Done: true,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := NewOllamaProvider(&ProviderConfig{
		Type:       ProviderOllama,
		OllamaHost: server.URL,
	})

	req := &SummarizeRequest{
		Content:   "A very long content that needs to be summarized...",
		MaxLength: 100,
	}

	resp, err := provider.Summarize(context.Background(), req)
	if err != nil {
		t.Fatalf("Summarize() error: %v", err)
	}

	if resp.Summary == "" {
		t.Error("Expected a summary")
	}
}

func TestOllamaProviderHTTPErrors(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		expectedError  error
		serverResponse string
	}{
		{
			name:           "Connection refused simulation",
			statusCode:     503,
			expectedError:  ErrProviderUnavailable,
			serverResponse: `{"error": "service unavailable"}`,
		},
		{
			name:           "Model not found",
			statusCode:     404,
			expectedError:  nil, // Generic error
			serverResponse: `{"error": "model not found"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			provider := NewOllamaProvider(&ProviderConfig{
				Type:       ProviderOllama,
				OllamaHost: server.URL,
			})

			req := &CompletionRequest{
				Messages: []Message{
					{Role: RoleUser, Content: "Hello"},
				},
			}

			_, err := provider.Complete(context.Background(), req)
			if err == nil {
				t.Error("Expected error, got nil")
			}

			if tt.expectedError != nil && err != tt.expectedError {
				t.Errorf("Expected error %v, got %v", tt.expectedError, err)
			}
		})
	}
}
