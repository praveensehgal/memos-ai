package llm

import (
	"context"
	"fmt"
	"log/slog"

	storepb "github.com/usememos/memos/proto/gen/store"
)

// ConfigManager handles loading and saving LLM configuration.
// It bridges the gap between proto-based storage and the runtime service.
type ConfigManager struct {
	service Service
}

// NewConfigManager creates a new configuration manager.
func NewConfigManager(service Service) *ConfigManager {
	return &ConfigManager{
		service: service,
	}
}

// LoadFromProto initializes the service from proto configuration.
// This should be called at startup to restore saved settings.
func (m *ConfigManager) LoadFromProto(ctx context.Context, setting *storepb.InstanceLLMSetting) error {
	if setting == nil {
		slog.Debug("No LLM settings found, using defaults")
		return nil
	}

	// Register providers based on their configuration
	if config := setting.GetOpenaiConfig(); config != nil {
		provider := NewOpenAIProviderFromProto(config)
		if err := m.service.RegisterProvider(provider); err != nil {
			slog.Warn("Failed to register OpenAI provider", slog.Any("error", err))
		}
	}

	if config := setting.GetOllamaConfig(); config != nil {
		provider := NewOllamaProviderFromProto(config)
		if err := m.service.RegisterProvider(provider); err != nil {
			slog.Warn("Failed to register Ollama provider", slog.Any("error", err))
		}
	}

	// Set the active provider if specified
	if setting.Provider != storepb.InstanceLLMSetting_LLM_PROVIDER_UNSPECIFIED {
		providerType := protoProviderToType(setting.Provider)
		if providerType != "" {
			if err := m.service.SetActiveProvider(providerType); err != nil {
				slog.Warn("Failed to set active provider from settings",
					slog.String("provider", string(providerType)),
					slog.Any("error", err))
				// Try to use fallback
				if err := m.tryFallbackProvider(ctx); err != nil {
					slog.Warn("No fallback provider available", slog.Any("error", err))
				}
			}
		}
	}

	return nil
}

// ToProto converts the current service state to proto configuration.
// This should be called when saving settings.
func (m *ConfigManager) ToProto() *storepb.InstanceLLMSetting {
	setting := &storepb.InstanceLLMSetting{}

	// Get all registered providers and their configurations
	providers := m.service.ListProviders()
	for _, status := range providers {
		switch status.Type {
		case ProviderOpenAI:
			if provider, err := m.service.GetProviderByType(ProviderOpenAI); err == nil {
				if openai, ok := provider.(*OpenAIProvider); ok {
					setting.OpenaiConfig = openai.ToProto()
				}
			}
		case ProviderOllama:
			if provider, err := m.service.GetProviderByType(ProviderOllama); err == nil {
				if ollama, ok := provider.(*OllamaProvider); ok {
					setting.OllamaConfig = ollama.ToProto()
				}
			}
		}

		// Set the active provider
		if status.Active {
			setting.Provider = typeToProtoProvider(status.Type)
		}
	}

	return setting
}

// SetActiveProviderWithFallback sets the active provider with automatic fallback.
// If the requested provider is not available, it tries to find an alternative.
func (m *ConfigManager) SetActiveProviderWithFallback(ctx context.Context, providerType ProviderType) error {
	// First try the requested provider
	if err := m.service.SetActiveProvider(providerType); err == nil {
		// Check if it's actually configured
		if provider, err := m.service.GetProviderByType(providerType); err == nil {
			if provider.IsConfigured(ctx) {
				return nil
			}
		}
	}

	// Provider not available, try fallback
	slog.Info("Requested provider not available, trying fallback",
		slog.String("requested", string(providerType)))

	return m.tryFallbackProvider(ctx)
}

// tryFallbackProvider attempts to select an available provider as fallback.
func (m *ConfigManager) tryFallbackProvider(ctx context.Context) error {
	providers := m.service.ListProviders()

	// Priority order for fallback: Ollama (local), OpenAI, Anthropic, Gemini
	fallbackOrder := []ProviderType{ProviderOllama, ProviderOpenAI, ProviderAnthropic, ProviderGemini}

	for _, providerType := range fallbackOrder {
		for _, status := range providers {
			if status.Type == providerType && status.Configured {
				if err := m.service.SetActiveProvider(providerType); err == nil {
					slog.Info("Fallback provider selected", slog.String("provider", string(providerType)))
					return nil
				}
			}
		}
	}

	return fmt.Errorf("no configured provider available for fallback")
}

// GetService returns the underlying service.
func (m *ConfigManager) GetService() Service {
	return m.service
}

// protoProviderToType converts proto provider enum to ProviderType.
func protoProviderToType(provider storepb.InstanceLLMSetting_LLMProvider) ProviderType {
	switch provider {
	case storepb.InstanceLLMSetting_OPENAI:
		return ProviderOpenAI
	case storepb.InstanceLLMSetting_ANTHROPIC:
		return ProviderAnthropic
	case storepb.InstanceLLMSetting_GEMINI:
		return ProviderGemini
	case storepb.InstanceLLMSetting_OLLAMA:
		return ProviderOllama
	default:
		return ""
	}
}

// typeToProtoProvider converts ProviderType to proto provider enum.
func typeToProtoProvider(providerType ProviderType) storepb.InstanceLLMSetting_LLMProvider {
	switch providerType {
	case ProviderOpenAI:
		return storepb.InstanceLLMSetting_OPENAI
	case ProviderAnthropic:
		return storepb.InstanceLLMSetting_ANTHROPIC
	case ProviderGemini:
		return storepb.InstanceLLMSetting_GEMINI
	case ProviderOllama:
		return storepb.InstanceLLMSetting_OLLAMA
	default:
		return storepb.InstanceLLMSetting_LLM_PROVIDER_UNSPECIFIED
	}
}
