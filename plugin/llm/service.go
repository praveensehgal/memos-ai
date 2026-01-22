package llm

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// Service manages multiple LLM providers and provides a unified interface.
// It handles provider selection, fallback, and configuration.
type Service interface {
	// GetProvider returns the currently active provider.
	GetProvider() Provider

	// GetProviderByType returns a specific provider by type.
	GetProviderByType(providerType ProviderType) (Provider, error)

	// SetActiveProvider sets the active provider.
	SetActiveProvider(providerType ProviderType) error

	// RegisterProvider adds a provider to the service.
	RegisterProvider(provider Provider) error

	// ListProviders returns all registered providers and their status.
	ListProviders() []ProviderStatus

	// IsConfigured checks if any provider is configured and ready.
	IsConfigured(ctx context.Context) bool

	// Complete performs a chat completion using the active provider.
	Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)

	// Embed generates embeddings using the active provider.
	Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error)

	// SuggestTags suggests tags using the active provider.
	SuggestTags(ctx context.Context, req *SuggestTagsRequest) (*SuggestTagsResponse, error)

	// Summarize generates a summary using the active provider.
	Summarize(ctx context.Context, req *SummarizeRequest) (*SummarizeResponse, error)
}

// ProviderStatus represents the status of a registered provider.
type ProviderStatus struct {
	// Type is the provider type.
	Type ProviderType `json:"type"`

	// Name is the human-readable name.
	Name string `json:"name"`

	// Configured indicates if the provider is properly configured.
	Configured bool `json:"configured"`

	// Active indicates if this is the currently active provider.
	Active bool `json:"active"`

	// DefaultModel is the default model for this provider.
	DefaultModel string `json:"default_model"`
}

// service implements the Service interface.
type service struct {
	mu             sync.RWMutex
	providers      map[ProviderType]Provider
	activeProvider ProviderType
}

// NewService creates a new LLM service.
func NewService() Service {
	return &service{
		providers: make(map[ProviderType]Provider),
	}
}

// GetProvider returns the currently active provider.
func (s *service) GetProvider() Provider {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.activeProvider == "" {
		return nil
	}

	return s.providers[s.activeProvider]
}

// GetProviderByType returns a specific provider by type.
func (s *service) GetProviderByType(providerType ProviderType) (Provider, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	provider, ok := s.providers[providerType]
	if !ok {
		return nil, fmt.Errorf("provider %s not registered", providerType)
	}

	return provider, nil
}

// SetActiveProvider sets the active provider.
func (s *service) SetActiveProvider(providerType ProviderType) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.providers[providerType]; !ok {
		return fmt.Errorf("provider %s not registered", providerType)
	}

	s.activeProvider = providerType
	slog.Info("LLM active provider changed", slog.String("provider", string(providerType)))

	return nil
}

// RegisterProvider adds a provider to the service.
func (s *service) RegisterProvider(provider Provider) error {
	if provider == nil {
		return fmt.Errorf("cannot register nil provider")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	providerType := provider.GetType()
	s.providers[providerType] = provider

	slog.Info("LLM provider registered",
		slog.String("provider", string(providerType)),
		slog.String("name", provider.GetName()))

	// Auto-select first configured provider as active
	if s.activeProvider == "" && provider.IsConfigured(context.Background()) {
		s.activeProvider = providerType
		slog.Info("LLM auto-selected active provider", slog.String("provider", string(providerType)))
	}

	return nil
}

// ListProviders returns all registered providers and their status.
func (s *service) ListProviders() []ProviderStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := context.Background()
	statuses := make([]ProviderStatus, 0, len(s.providers))

	for providerType, provider := range s.providers {
		statuses = append(statuses, ProviderStatus{
			Type:         providerType,
			Name:         provider.GetName(),
			Configured:   provider.IsConfigured(ctx),
			Active:       providerType == s.activeProvider,
			DefaultModel: provider.GetDefaultModel(),
		})
	}

	return statuses
}

// IsConfigured checks if any provider is configured and ready.
func (s *service) IsConfigured(ctx context.Context) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, provider := range s.providers {
		if provider.IsConfigured(ctx) {
			return true
		}
	}

	return false
}

// Complete performs a chat completion using the active provider.
func (s *service) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	provider := s.GetProvider()
	if provider == nil {
		return nil, ErrProviderNotConfigured
	}

	if !provider.IsConfigured(ctx) {
		return nil, ErrProviderNotConfigured
	}

	return provider.Complete(ctx, req)
}

// Embed generates embeddings using the active provider.
func (s *service) Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	provider := s.GetProvider()
	if provider == nil {
		return nil, ErrProviderNotConfigured
	}

	if !provider.IsConfigured(ctx) {
		return nil, ErrProviderNotConfigured
	}

	return provider.Embed(ctx, req)
}

// SuggestTags suggests tags using the active provider.
func (s *service) SuggestTags(ctx context.Context, req *SuggestTagsRequest) (*SuggestTagsResponse, error) {
	provider := s.GetProvider()
	if provider == nil {
		return nil, ErrProviderNotConfigured
	}

	if !provider.IsConfigured(ctx) {
		return nil, ErrProviderNotConfigured
	}

	return provider.SuggestTags(ctx, req)
}

// Summarize generates a summary using the active provider.
func (s *service) Summarize(ctx context.Context, req *SummarizeRequest) (*SummarizeResponse, error) {
	provider := s.GetProvider()
	if provider == nil {
		return nil, ErrProviderNotConfigured
	}

	if !provider.IsConfigured(ctx) {
		return nil, ErrProviderNotConfigured
	}

	return provider.Summarize(ctx, req)
}
