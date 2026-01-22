package llm

import (
	"context"
	"testing"

	storepb "github.com/usememos/memos/proto/gen/store"
)

func TestNewConfigManager(t *testing.T) {
	service := NewService()
	manager := NewConfigManager(service)

	if manager == nil {
		t.Fatal("NewConfigManager returned nil")
	}

	if manager.GetService() != service {
		t.Error("GetService() should return the same service")
	}
}

func TestConfigManager_LoadFromProto_Nil(t *testing.T) {
	service := NewService()
	manager := NewConfigManager(service)

	err := manager.LoadFromProto(context.Background(), nil)
	if err != nil {
		t.Errorf("LoadFromProto with nil should not error: %v", err)
	}
}

func TestConfigManager_LoadFromProto_OpenAI(t *testing.T) {
	service := NewService()
	manager := NewConfigManager(service)

	setting := &storepb.InstanceLLMSetting{
		Provider: storepb.InstanceLLMSetting_OPENAI,
		OpenaiConfig: &storepb.LLMOpenAIConfig{
			ApiKey:         "test-api-key",
			BaseUrl:        "https://api.test.com",
			DefaultModel:   "gpt-4",
			EmbeddingModel: "text-embedding-ada-002",
		},
	}

	err := manager.LoadFromProto(context.Background(), setting)
	if err != nil {
		t.Errorf("LoadFromProto failed: %v", err)
	}

	// Verify provider was registered
	providers := service.ListProviders()
	found := false
	for _, p := range providers {
		if p.Type == ProviderOpenAI {
			found = true
			if !p.Active {
				t.Error("OpenAI should be active")
			}
			break
		}
	}
	if !found {
		t.Error("OpenAI provider should be registered")
	}
}

func TestConfigManager_LoadFromProto_Ollama(t *testing.T) {
	service := NewService()
	manager := NewConfigManager(service)

	setting := &storepb.InstanceLLMSetting{
		Provider: storepb.InstanceLLMSetting_OLLAMA,
		OllamaConfig: &storepb.LLMOllamaConfig{
			Host:           "http://localhost:11434",
			DefaultModel:   "llama2",
			EmbeddingModel: "nomic-embed-text",
		},
	}

	err := manager.LoadFromProto(context.Background(), setting)
	if err != nil {
		t.Errorf("LoadFromProto failed: %v", err)
	}

	// Verify provider was registered
	providers := service.ListProviders()
	found := false
	for _, p := range providers {
		if p.Type == ProviderOllama {
			found = true
			if !p.Active {
				t.Error("Ollama should be active")
			}
			break
		}
	}
	if !found {
		t.Error("Ollama provider should be registered")
	}
}

func TestConfigManager_LoadFromProto_MultipleProviders(t *testing.T) {
	service := NewService()
	manager := NewConfigManager(service)

	setting := &storepb.InstanceLLMSetting{
		Provider: storepb.InstanceLLMSetting_OPENAI,
		OpenaiConfig: &storepb.LLMOpenAIConfig{
			ApiKey: "test-api-key",
		},
		OllamaConfig: &storepb.LLMOllamaConfig{
			Host: "http://localhost:11434",
		},
	}

	err := manager.LoadFromProto(context.Background(), setting)
	if err != nil {
		t.Errorf("LoadFromProto failed: %v", err)
	}

	// Verify both providers were registered
	providers := service.ListProviders()
	openaiFound := false
	ollamaFound := false
	for _, p := range providers {
		if p.Type == ProviderOpenAI {
			openaiFound = true
			if !p.Active {
				t.Error("OpenAI should be active (it was specified)")
			}
		}
		if p.Type == ProviderOllama {
			ollamaFound = true
			if p.Active {
				t.Error("Ollama should not be active")
			}
		}
	}
	if !openaiFound {
		t.Error("OpenAI provider should be registered")
	}
	if !ollamaFound {
		t.Error("Ollama provider should be registered")
	}
}

func TestConfigManager_ToProto_Empty(t *testing.T) {
	service := NewService()
	manager := NewConfigManager(service)

	setting := manager.ToProto()
	if setting == nil {
		t.Fatal("ToProto returned nil")
	}

	// Empty service should return empty setting
	if setting.Provider != storepb.InstanceLLMSetting_LLM_PROVIDER_UNSPECIFIED {
		t.Errorf("Empty service should have unspecified provider, got %v", setting.Provider)
	}
}

func TestConfigManager_ToProto_OpenAI(t *testing.T) {
	service := NewService()
	manager := NewConfigManager(service)

	// Register an OpenAI provider
	provider := NewOpenAIProvider(&ProviderConfig{
		Type:           ProviderOpenAI,
		APIKey:         "test-key",
		BaseURL:        "https://api.test.com",
		DefaultModel:   "gpt-4",
		EmbeddingModel: "text-embedding-3-small",
	})
	_ = service.RegisterProvider(provider)

	setting := manager.ToProto()
	if setting == nil {
		t.Fatal("ToProto returned nil")
	}

	if setting.Provider != storepb.InstanceLLMSetting_OPENAI {
		t.Errorf("Expected OPENAI provider, got %v", setting.Provider)
	}

	if setting.OpenaiConfig == nil {
		t.Fatal("OpenAI config should not be nil")
	}

	if setting.OpenaiConfig.ApiKey != "test-key" {
		t.Errorf("Expected api key 'test-key', got %s", setting.OpenaiConfig.ApiKey)
	}

	if setting.OpenaiConfig.BaseUrl != "https://api.test.com" {
		t.Errorf("Expected base url 'https://api.test.com', got %s", setting.OpenaiConfig.BaseUrl)
	}

	if setting.OpenaiConfig.DefaultModel != "gpt-4" {
		t.Errorf("Expected default model 'gpt-4', got %s", setting.OpenaiConfig.DefaultModel)
	}

	if setting.OpenaiConfig.EmbeddingModel != "text-embedding-3-small" {
		t.Errorf("Expected embedding model 'text-embedding-3-small', got %s", setting.OpenaiConfig.EmbeddingModel)
	}
}

func TestConfigManager_ToProto_Ollama(t *testing.T) {
	service := NewService()
	manager := NewConfigManager(service)

	// Register an Ollama provider
	provider := NewOllamaProvider(&ProviderConfig{
		Type:           ProviderOllama,
		OllamaHost:     "http://localhost:11434",
		DefaultModel:   "llama2",
		EmbeddingModel: "nomic-embed-text",
	})
	_ = service.RegisterProvider(provider)

	setting := manager.ToProto()
	if setting == nil {
		t.Fatal("ToProto returned nil")
	}

	if setting.Provider != storepb.InstanceLLMSetting_OLLAMA {
		t.Errorf("Expected OLLAMA provider, got %v", setting.Provider)
	}

	if setting.OllamaConfig == nil {
		t.Fatal("Ollama config should not be nil")
	}

	if setting.OllamaConfig.Host != "http://localhost:11434" {
		t.Errorf("Expected host 'http://localhost:11434', got %s", setting.OllamaConfig.Host)
	}

	if setting.OllamaConfig.DefaultModel != "llama2" {
		t.Errorf("Expected default model 'llama2', got %s", setting.OllamaConfig.DefaultModel)
	}

	if setting.OllamaConfig.EmbeddingModel != "nomic-embed-text" {
		t.Errorf("Expected embedding model 'nomic-embed-text', got %s", setting.OllamaConfig.EmbeddingModel)
	}
}

func TestConfigManager_RoundTrip(t *testing.T) {
	// Create first service and manager
	service1 := NewService()
	manager1 := NewConfigManager(service1)

	// Register providers
	openaiProvider := NewOpenAIProvider(&ProviderConfig{
		Type:           ProviderOpenAI,
		APIKey:         "test-api-key",
		BaseURL:        "https://api.openai.com/v1",
		DefaultModel:   "gpt-4o-mini",
		EmbeddingModel: "text-embedding-3-small",
	})
	ollamaProvider := NewOllamaProvider(&ProviderConfig{
		Type:           ProviderOllama,
		OllamaHost:     "http://localhost:11434",
		DefaultModel:   "llama3.2",
		EmbeddingModel: "nomic-embed-text",
	})
	_ = service1.RegisterProvider(openaiProvider)
	_ = service1.RegisterProvider(ollamaProvider)

	// Set OpenAI as active
	_ = service1.SetActiveProvider(ProviderOpenAI)

	// Convert to proto
	setting := manager1.ToProto()

	// Create second service and manager
	service2 := NewService()
	manager2 := NewConfigManager(service2)

	// Load from proto
	err := manager2.LoadFromProto(context.Background(), setting)
	if err != nil {
		t.Fatalf("LoadFromProto failed: %v", err)
	}

	// Verify providers match
	providers1 := service1.ListProviders()
	providers2 := service2.ListProviders()

	if len(providers1) != len(providers2) {
		t.Errorf("Provider count mismatch: %d vs %d", len(providers1), len(providers2))
	}

	// Verify active provider
	var active1, active2 ProviderType
	for _, p := range providers1 {
		if p.Active {
			active1 = p.Type
		}
	}
	for _, p := range providers2 {
		if p.Active {
			active2 = p.Type
		}
	}
	if active1 != active2 {
		t.Errorf("Active provider mismatch: %s vs %s", active1, active2)
	}
}

func TestProtoProviderToType(t *testing.T) {
	tests := []struct {
		proto    storepb.InstanceLLMSetting_LLMProvider
		expected ProviderType
	}{
		{storepb.InstanceLLMSetting_OPENAI, ProviderOpenAI},
		{storepb.InstanceLLMSetting_ANTHROPIC, ProviderAnthropic},
		{storepb.InstanceLLMSetting_GEMINI, ProviderGemini},
		{storepb.InstanceLLMSetting_OLLAMA, ProviderOllama},
		{storepb.InstanceLLMSetting_LLM_PROVIDER_UNSPECIFIED, ""},
	}

	for _, tc := range tests {
		result := protoProviderToType(tc.proto)
		if result != tc.expected {
			t.Errorf("protoProviderToType(%v) = %s, expected %s", tc.proto, result, tc.expected)
		}
	}
}

func TestTypeToProtoProvider(t *testing.T) {
	tests := []struct {
		providerType ProviderType
		expected     storepb.InstanceLLMSetting_LLMProvider
	}{
		{ProviderOpenAI, storepb.InstanceLLMSetting_OPENAI},
		{ProviderAnthropic, storepb.InstanceLLMSetting_ANTHROPIC},
		{ProviderGemini, storepb.InstanceLLMSetting_GEMINI},
		{ProviderOllama, storepb.InstanceLLMSetting_OLLAMA},
		{"unknown", storepb.InstanceLLMSetting_LLM_PROVIDER_UNSPECIFIED},
	}

	for _, tc := range tests {
		result := typeToProtoProvider(tc.providerType)
		if result != tc.expected {
			t.Errorf("typeToProtoProvider(%s) = %v, expected %v", tc.providerType, result, tc.expected)
		}
	}
}

func TestConfigManager_SetActiveProviderWithFallback(t *testing.T) {
	service := NewService()
	manager := NewConfigManager(service)

	// Register only Ollama provider
	ollamaProvider := NewOllamaProvider(&ProviderConfig{
		Type:       ProviderOllama,
		OllamaHost: "http://localhost:11434",
	})
	_ = service.RegisterProvider(ollamaProvider)

	// Try to set OpenAI as active (not registered), should fallback to Ollama
	err := manager.SetActiveProviderWithFallback(context.Background(), ProviderOpenAI)
	if err != nil {
		t.Errorf("SetActiveProviderWithFallback should succeed with fallback: %v", err)
	}

	// Verify Ollama is now active
	providers := service.ListProviders()
	for _, p := range providers {
		if p.Type == ProviderOllama && !p.Active {
			t.Error("Ollama should be active after fallback")
		}
	}
}

func TestConfigManager_SetActiveProviderWithFallback_NoFallback(t *testing.T) {
	service := NewService()
	manager := NewConfigManager(service)

	// No providers registered, should fail
	err := manager.SetActiveProviderWithFallback(context.Background(), ProviderOpenAI)
	if err == nil {
		t.Error("SetActiveProviderWithFallback should fail with no available providers")
	}
}

func TestConfigManager_TryFallbackProvider_Priority(t *testing.T) {
	service := NewService()
	manager := NewConfigManager(service)

	// Register OpenAI first, then Ollama
	openaiProvider := NewOpenAIProvider(&ProviderConfig{
		Type:   ProviderOpenAI,
		APIKey: "test-key",
	})
	ollamaProvider := NewOllamaProvider(&ProviderConfig{
		Type:       ProviderOllama,
		OllamaHost: "http://localhost:11434",
	})
	_ = service.RegisterProvider(openaiProvider)
	_ = service.RegisterProvider(ollamaProvider)

	// Clear any active provider and try fallback
	// Ollama should be selected first due to priority order (local first)
	err := manager.tryFallbackProvider(context.Background())
	if err != nil {
		t.Errorf("tryFallbackProvider failed: %v", err)
	}

	// Verify Ollama is active (it has higher fallback priority as local provider)
	providers := service.ListProviders()
	for _, p := range providers {
		if p.Type == ProviderOllama && !p.Active {
			t.Error("Ollama should be active (highest fallback priority)")
		}
	}
}
