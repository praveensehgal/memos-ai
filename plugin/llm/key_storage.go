package llm

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

var (
	// ErrKeyNotFound indicates the API key was not found.
	ErrKeyNotFound = errors.New("API key not found")

	// ErrKeyAlreadyExists indicates an API key already exists for this provider.
	ErrKeyAlreadyExists = errors.New("API key already exists for this provider")
)

// StoredAPIKey represents an encrypted API key stored in the system.
type StoredAPIKey struct {
	// ID is a unique identifier for this key (derived from key hash).
	ID string `json:"id"`

	// ProviderType is the LLM provider this key is for.
	ProviderType ProviderType `json:"provider_type"`

	// EncryptedKey is the AES-256-GCM encrypted API key.
	EncryptedKey string `json:"encrypted_key"`

	// MaskedKey is the masked version for display (e.g., "****abcd").
	MaskedKey string `json:"masked_key"`

	// CreatedAt is when the key was stored.
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the key was last updated.
	UpdatedAt time.Time `json:"updated_at"`

	// LastUsedAt is when the key was last used for an API call.
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`

	// UserID is the ID of the user who owns this key (0 for instance-level keys).
	UserID int32 `json:"user_id"`
}

// KeyStorageService manages API key storage with encryption.
type KeyStorageService interface {
	// StoreKey stores an API key with encryption.
	StoreKey(ctx context.Context, userID int32, providerType ProviderType, apiKey string) (*StoredAPIKey, error)

	// GetKey retrieves and decrypts an API key.
	GetKey(ctx context.Context, userID int32, providerType ProviderType) (string, error)

	// GetStoredKey retrieves the stored key metadata (without decrypting).
	GetStoredKey(ctx context.Context, userID int32, providerType ProviderType) (*StoredAPIKey, error)

	// UpdateKey updates an existing API key.
	UpdateKey(ctx context.Context, userID int32, providerType ProviderType, apiKey string) (*StoredAPIKey, error)

	// DeleteKey removes an API key.
	DeleteKey(ctx context.Context, userID int32, providerType ProviderType) error

	// ListKeys returns all stored keys for a user (without decrypting).
	ListKeys(ctx context.Context, userID int32) ([]*StoredAPIKey, error)

	// HasKey checks if a key exists for a provider.
	HasKey(ctx context.Context, userID int32, providerType ProviderType) bool

	// MarkKeyUsed updates the LastUsedAt timestamp.
	MarkKeyUsed(ctx context.Context, userID int32, providerType ProviderType) error
}

// InMemoryKeyStorage is an in-memory implementation of KeyStorageService.
// This is useful for testing and development. For production, use a database-backed implementation.
type InMemoryKeyStorage struct {
	crypto *KeyCrypto
	keys   map[string]*StoredAPIKey // key: "userID:providerType"
	mu     sync.RWMutex
}

// NewInMemoryKeyStorage creates a new in-memory key storage service.
func NewInMemoryKeyStorage(masterKey string) (*InMemoryKeyStorage, error) {
	crypto, err := NewKeyCrypto(masterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create crypto: %w", err)
	}

	return &InMemoryKeyStorage{
		crypto: crypto,
		keys:   make(map[string]*StoredAPIKey),
	}, nil
}

// storageKey generates a unique storage key for a user and provider.
func storageKey(userID int32, providerType ProviderType) string {
	return fmt.Sprintf("%d:%s", userID, providerType)
}

// StoreKey stores an API key with encryption.
func (s *InMemoryKeyStorage) StoreKey(ctx context.Context, userID int32, providerType ProviderType, apiKey string) (*StoredAPIKey, error) {
	// Validate the API key format
	if err := ValidateAPIKeyFormat(providerType, apiKey); err != nil {
		return nil, fmt.Errorf("invalid API key: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := storageKey(userID, providerType)
	if _, exists := s.keys[key]; exists {
		return nil, ErrKeyAlreadyExists
	}

	// Encrypt the key
	encryptedKey, err := s.crypto.Encrypt(apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt key: %w", err)
	}

	now := time.Now()
	stored := &StoredAPIKey{
		ID:           GenerateKeyID(apiKey),
		ProviderType: providerType,
		EncryptedKey: encryptedKey,
		MaskedKey:    MaskAPIKey(apiKey),
		CreatedAt:    now,
		UpdatedAt:    now,
		UserID:       userID,
	}

	s.keys[key] = stored

	slog.Info("API key stored",
		slog.Int("user_id", int(userID)),
		slog.String("provider", string(providerType)),
		slog.String("key_id", stored.ID))

	return stored, nil
}

// GetKey retrieves and decrypts an API key.
func (s *InMemoryKeyStorage) GetKey(ctx context.Context, userID int32, providerType ProviderType) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := storageKey(userID, providerType)
	stored, exists := s.keys[key]
	if !exists {
		return "", ErrKeyNotFound
	}

	// Decrypt the key
	apiKey, err := s.crypto.Decrypt(stored.EncryptedKey)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt key: %w", err)
	}

	return apiKey, nil
}

// GetStoredKey retrieves the stored key metadata (without decrypting).
func (s *InMemoryKeyStorage) GetStoredKey(ctx context.Context, userID int32, providerType ProviderType) (*StoredAPIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := storageKey(userID, providerType)
	stored, exists := s.keys[key]
	if !exists {
		return nil, ErrKeyNotFound
	}

	// Return a copy to prevent modification
	copy := *stored
	return &copy, nil
}

// UpdateKey updates an existing API key.
func (s *InMemoryKeyStorage) UpdateKey(ctx context.Context, userID int32, providerType ProviderType, apiKey string) (*StoredAPIKey, error) {
	// Validate the API key format
	if err := ValidateAPIKeyFormat(providerType, apiKey); err != nil {
		return nil, fmt.Errorf("invalid API key: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := storageKey(userID, providerType)
	stored, exists := s.keys[key]
	if !exists {
		return nil, ErrKeyNotFound
	}

	// Encrypt the new key
	encryptedKey, err := s.crypto.Encrypt(apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt key: %w", err)
	}

	// Update the stored key
	stored.ID = GenerateKeyID(apiKey)
	stored.EncryptedKey = encryptedKey
	stored.MaskedKey = MaskAPIKey(apiKey)
	stored.UpdatedAt = time.Now()

	slog.Info("API key updated",
		slog.Int("user_id", int(userID)),
		slog.String("provider", string(providerType)),
		slog.String("key_id", stored.ID))

	// Return a copy
	copy := *stored
	return &copy, nil
}

// DeleteKey removes an API key.
func (s *InMemoryKeyStorage) DeleteKey(ctx context.Context, userID int32, providerType ProviderType) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := storageKey(userID, providerType)
	if _, exists := s.keys[key]; !exists {
		return ErrKeyNotFound
	}

	delete(s.keys, key)

	slog.Info("API key deleted",
		slog.Int("user_id", int(userID)),
		slog.String("provider", string(providerType)))

	return nil
}

// ListKeys returns all stored keys for a user (without decrypting).
func (s *InMemoryKeyStorage) ListKeys(ctx context.Context, userID int32) ([]*StoredAPIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*StoredAPIKey
	prefix := fmt.Sprintf("%d:", userID)

	for key, stored := range s.keys {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			// Return a copy
			copy := *stored
			result = append(result, &copy)
		}
	}

	return result, nil
}

// HasKey checks if a key exists for a provider.
func (s *InMemoryKeyStorage) HasKey(ctx context.Context, userID int32, providerType ProviderType) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := storageKey(userID, providerType)
	_, exists := s.keys[key]
	return exists
}

// MarkKeyUsed updates the LastUsedAt timestamp.
func (s *InMemoryKeyStorage) MarkKeyUsed(ctx context.Context, userID int32, providerType ProviderType) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := storageKey(userID, providerType)
	stored, exists := s.keys[key]
	if !exists {
		return ErrKeyNotFound
	}

	now := time.Now()
	stored.LastUsedAt = &now

	return nil
}

// Ensure InMemoryKeyStorage implements KeyStorageService.
var _ KeyStorageService = (*InMemoryKeyStorage)(nil)
