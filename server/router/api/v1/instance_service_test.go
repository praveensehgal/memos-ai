package v1

import (
	"testing"

	storepb "github.com/usememos/memos/proto/gen/store"
)

func TestPreserveExistingAPIKeys_OpenAI(t *testing.T) {
	existing := &storepb.InstanceLLMSetting{
		OpenaiConfig: &storepb.LLMOpenAIConfig{
			ApiKey:       "sk-existing-key-123",
			BaseUrl:      "https://api.openai.com/v1",
			DefaultModel: "gpt-4",
		},
	}

	// Test case 1: Empty API key should be preserved
	newSetting := &storepb.InstanceLLMSetting{
		OpenaiConfig: &storepb.LLMOpenAIConfig{
			ApiKey:       "", // Empty - should preserve existing
			BaseUrl:      "https://custom.api.com",
			DefaultModel: "gpt-4o",
		},
	}

	preserveExistingAPIKeys(newSetting, existing)

	if newSetting.OpenaiConfig.ApiKey != "sk-existing-key-123" {
		t.Errorf("Expected API key to be preserved, got %s", newSetting.OpenaiConfig.ApiKey)
	}
	if newSetting.OpenaiConfig.BaseUrl != "https://custom.api.com" {
		t.Errorf("Expected BaseUrl to be updated, got %s", newSetting.OpenaiConfig.BaseUrl)
	}

	// Test case 2: Masked API key should be preserved
	newSetting2 := &storepb.InstanceLLMSetting{
		OpenaiConfig: &storepb.LLMOpenAIConfig{
			ApiKey:       maskedAPIKey, // Masked - should preserve existing
			BaseUrl:      "https://api.openai.com/v1",
			DefaultModel: "gpt-4",
		},
	}

	preserveExistingAPIKeys(newSetting2, existing)

	if newSetting2.OpenaiConfig.ApiKey != "sk-existing-key-123" {
		t.Errorf("Expected masked API key to be preserved, got %s", newSetting2.OpenaiConfig.ApiKey)
	}

	// Test case 3: New API key should NOT be preserved
	newSetting3 := &storepb.InstanceLLMSetting{
		OpenaiConfig: &storepb.LLMOpenAIConfig{
			ApiKey:       "sk-new-key-456", // New key - should NOT be overwritten
			BaseUrl:      "https://api.openai.com/v1",
			DefaultModel: "gpt-4",
		},
	}

	preserveExistingAPIKeys(newSetting3, existing)

	if newSetting3.OpenaiConfig.ApiKey != "sk-new-key-456" {
		t.Errorf("Expected new API key to be kept, got %s", newSetting3.OpenaiConfig.ApiKey)
	}
}

func TestPreserveExistingAPIKeys_Anthropic(t *testing.T) {
	existing := &storepb.InstanceLLMSetting{
		AnthropicConfig: &storepb.LLMAnthropicConfig{
			ApiKey:       "sk-ant-existing-123",
			BaseUrl:      "https://api.anthropic.com",
			DefaultModel: "claude-3-5-sonnet",
		},
	}

	newSetting := &storepb.InstanceLLMSetting{
		AnthropicConfig: &storepb.LLMAnthropicConfig{
			ApiKey:       "", // Empty - should preserve existing
			BaseUrl:      "https://custom.anthropic.com",
			DefaultModel: "claude-3-opus",
		},
	}

	preserveExistingAPIKeys(newSetting, existing)

	if newSetting.AnthropicConfig.ApiKey != "sk-ant-existing-123" {
		t.Errorf("Expected Anthropic API key to be preserved, got %s", newSetting.AnthropicConfig.ApiKey)
	}
}

func TestPreserveExistingAPIKeys_NoExistingConfig(t *testing.T) {
	// Edge case: existing has no config for a provider
	existing := &storepb.InstanceLLMSetting{
		OpenaiConfig: nil, // No existing config
	}

	newSetting := &storepb.InstanceLLMSetting{
		OpenaiConfig: &storepb.LLMOpenAIConfig{
			ApiKey:       "",
			BaseUrl:      "https://api.openai.com/v1",
			DefaultModel: "gpt-4",
		},
	}

	// This should not panic
	preserveExistingAPIKeys(newSetting, existing)

	// API key should remain empty since there's no existing config
	if newSetting.OpenaiConfig.ApiKey != "" {
		t.Errorf("Expected API key to remain empty when no existing config, got %s", newSetting.OpenaiConfig.ApiKey)
	}
}
