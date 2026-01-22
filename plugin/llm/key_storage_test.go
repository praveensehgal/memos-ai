package llm

import (
	"context"
	"testing"
)

func TestNewInMemoryKeyStorage(t *testing.T) {
	tests := []struct {
		name      string
		masterKey string
		wantErr   bool
	}{
		{
			name:      "valid master key",
			masterKey: "test-master-key-12345",
			wantErr:   false,
		},
		{
			name:      "master key too short",
			masterKey: "short",
			wantErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			storage, err := NewInMemoryKeyStorage(tc.masterKey)
			if tc.wantErr {
				if err == nil {
					t.Error("NewInMemoryKeyStorage() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("NewInMemoryKeyStorage() unexpected error: %v", err)
				}
				if storage == nil {
					t.Error("NewInMemoryKeyStorage() returned nil storage")
				}
			}
		})
	}
}

func TestKeyStorage_StoreKey(t *testing.T) {
	storage, err := NewInMemoryKeyStorage("test-master-key-12345")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	ctx := context.Background()

	tests := []struct {
		name         string
		userID       int32
		providerType ProviderType
		apiKey       string
		wantErr      bool
		errType      error
	}{
		{
			name:         "store OpenAI key",
			userID:       1,
			providerType: ProviderOpenAI,
			apiKey:       "sk-valid-openai-key-12345678901234567890",
			wantErr:      false,
		},
		{
			name:         "store Anthropic key",
			userID:       1,
			providerType: ProviderAnthropic,
			apiKey:       "sk-ant-api03-valid-key-12345",
			wantErr:      false,
		},
		{
			name:         "store Ollama key (any key allowed)",
			userID:       1,
			providerType: ProviderOllama,
			apiKey:       "any-ollama-key",
			wantErr:      false,
		},
		{
			name:         "invalid OpenAI key format",
			userID:       2,
			providerType: ProviderOpenAI,
			apiKey:       "invalid-key",
			wantErr:      true,
		},
		{
			name:         "empty key",
			userID:       2,
			providerType: ProviderOpenAI,
			apiKey:       "",
			wantErr:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stored, err := storage.StoreKey(ctx, tc.userID, tc.providerType, tc.apiKey)
			if tc.wantErr {
				if err == nil {
					t.Error("StoreKey() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("StoreKey() unexpected error: %v", err)
				return
			}

			// Verify stored key metadata
			if stored.ID == "" {
				t.Error("StoreKey() returned key with empty ID")
			}
			if stored.ProviderType != tc.providerType {
				t.Errorf("StoreKey() provider type = %v, want %v", stored.ProviderType, tc.providerType)
			}
			if stored.MaskedKey == tc.apiKey {
				t.Error("StoreKey() masked key should not equal original key")
			}
			if stored.EncryptedKey == tc.apiKey {
				t.Error("StoreKey() encrypted key should not equal original key")
			}
			if stored.UserID != tc.userID {
				t.Errorf("StoreKey() user ID = %v, want %v", stored.UserID, tc.userID)
			}
		})
	}
}

func TestKeyStorage_StoreKey_DuplicateError(t *testing.T) {
	storage, _ := NewInMemoryKeyStorage("test-master-key-12345")
	ctx := context.Background()

	// Store first key
	_, err := storage.StoreKey(ctx, 1, ProviderOpenAI, "sk-first-key-123456789012345678901234567890")
	if err != nil {
		t.Fatalf("First StoreKey() failed: %v", err)
	}

	// Try to store duplicate
	_, err = storage.StoreKey(ctx, 1, ProviderOpenAI, "sk-second-key-12345678901234567890123456789")
	if err != ErrKeyAlreadyExists {
		t.Errorf("Second StoreKey() error = %v, want ErrKeyAlreadyExists", err)
	}
}

func TestKeyStorage_GetKey(t *testing.T) {
	storage, _ := NewInMemoryKeyStorage("test-master-key-12345")
	ctx := context.Background()

	originalKey := "sk-test-api-key-1234567890123456789012345678"

	// Store a key
	_, err := storage.StoreKey(ctx, 1, ProviderOpenAI, originalKey)
	if err != nil {
		t.Fatalf("StoreKey() failed: %v", err)
	}

	// Retrieve the key
	retrievedKey, err := storage.GetKey(ctx, 1, ProviderOpenAI)
	if err != nil {
		t.Fatalf("GetKey() error: %v", err)
	}

	if retrievedKey != originalKey {
		t.Errorf("GetKey() = %v, want %v", retrievedKey, originalKey)
	}
}

func TestKeyStorage_GetKey_NotFound(t *testing.T) {
	storage, _ := NewInMemoryKeyStorage("test-master-key-12345")
	ctx := context.Background()

	_, err := storage.GetKey(ctx, 1, ProviderOpenAI)
	if err != ErrKeyNotFound {
		t.Errorf("GetKey() error = %v, want ErrKeyNotFound", err)
	}
}

func TestKeyStorage_GetStoredKey(t *testing.T) {
	storage, _ := NewInMemoryKeyStorage("test-master-key-12345")
	ctx := context.Background()

	originalKey := "sk-test-api-key-1234567890123456789012345678"

	// Store a key
	storedOnCreate, err := storage.StoreKey(ctx, 1, ProviderOpenAI, originalKey)
	if err != nil {
		t.Fatalf("StoreKey() failed: %v", err)
	}

	// Get stored key metadata
	storedKey, err := storage.GetStoredKey(ctx, 1, ProviderOpenAI)
	if err != nil {
		t.Fatalf("GetStoredKey() error: %v", err)
	}

	// Verify metadata matches
	if storedKey.ID != storedOnCreate.ID {
		t.Errorf("GetStoredKey() ID mismatch")
	}
	if storedKey.ProviderType != ProviderOpenAI {
		t.Errorf("GetStoredKey() provider type mismatch")
	}
	if storedKey.MaskedKey != MaskAPIKey(originalKey) {
		t.Errorf("GetStoredKey() masked key mismatch")
	}
}

func TestKeyStorage_UpdateKey(t *testing.T) {
	storage, _ := NewInMemoryKeyStorage("test-master-key-12345")
	ctx := context.Background()

	originalKey := "sk-original-key-123456789012345678901234567"
	updatedKey := "sk-updated-key-9876543210987654321098765432"

	// Store initial key
	_, err := storage.StoreKey(ctx, 1, ProviderOpenAI, originalKey)
	if err != nil {
		t.Fatalf("StoreKey() failed: %v", err)
	}

	// Update the key
	updated, err := storage.UpdateKey(ctx, 1, ProviderOpenAI, updatedKey)
	if err != nil {
		t.Fatalf("UpdateKey() error: %v", err)
	}

	// Verify update
	if updated.MaskedKey == MaskAPIKey(originalKey) {
		t.Error("UpdateKey() masked key should be updated")
	}
	if updated.MaskedKey != MaskAPIKey(updatedKey) {
		t.Error("UpdateKey() masked key mismatch")
	}

	// Verify retrieval returns updated key
	retrieved, err := storage.GetKey(ctx, 1, ProviderOpenAI)
	if err != nil {
		t.Fatalf("GetKey() error: %v", err)
	}
	if retrieved != updatedKey {
		t.Errorf("GetKey() after update = %v, want %v", retrieved, updatedKey)
	}
}

func TestKeyStorage_UpdateKey_NotFound(t *testing.T) {
	storage, _ := NewInMemoryKeyStorage("test-master-key-12345")
	ctx := context.Background()

	_, err := storage.UpdateKey(ctx, 1, ProviderOpenAI, "sk-new-key-123456789012345678901234567890")
	if err != ErrKeyNotFound {
		t.Errorf("UpdateKey() error = %v, want ErrKeyNotFound", err)
	}
}

func TestKeyStorage_UpdateKey_InvalidKey(t *testing.T) {
	storage, _ := NewInMemoryKeyStorage("test-master-key-12345")
	ctx := context.Background()

	// Store initial key
	_, err := storage.StoreKey(ctx, 1, ProviderOpenAI, "sk-valid-key-123456789012345678901234567890")
	if err != nil {
		t.Fatalf("StoreKey() failed: %v", err)
	}

	// Try to update with invalid key
	_, err = storage.UpdateKey(ctx, 1, ProviderOpenAI, "invalid-key")
	if err == nil {
		t.Error("UpdateKey() should fail with invalid key format")
	}
}

func TestKeyStorage_DeleteKey(t *testing.T) {
	storage, _ := NewInMemoryKeyStorage("test-master-key-12345")
	ctx := context.Background()

	// Store a key
	_, err := storage.StoreKey(ctx, 1, ProviderOpenAI, "sk-test-key-123456789012345678901234567890")
	if err != nil {
		t.Fatalf("StoreKey() failed: %v", err)
	}

	// Verify key exists
	if !storage.HasKey(ctx, 1, ProviderOpenAI) {
		t.Fatal("HasKey() should return true after store")
	}

	// Delete the key
	err = storage.DeleteKey(ctx, 1, ProviderOpenAI)
	if err != nil {
		t.Fatalf("DeleteKey() error: %v", err)
	}

	// Verify key is gone
	if storage.HasKey(ctx, 1, ProviderOpenAI) {
		t.Error("HasKey() should return false after delete")
	}

	// Try to get deleted key
	_, err = storage.GetKey(ctx, 1, ProviderOpenAI)
	if err != ErrKeyNotFound {
		t.Errorf("GetKey() after delete error = %v, want ErrKeyNotFound", err)
	}
}

func TestKeyStorage_DeleteKey_NotFound(t *testing.T) {
	storage, _ := NewInMemoryKeyStorage("test-master-key-12345")
	ctx := context.Background()

	err := storage.DeleteKey(ctx, 1, ProviderOpenAI)
	if err != ErrKeyNotFound {
		t.Errorf("DeleteKey() error = %v, want ErrKeyNotFound", err)
	}
}

func TestKeyStorage_ListKeys(t *testing.T) {
	storage, _ := NewInMemoryKeyStorage("test-master-key-12345")
	ctx := context.Background()

	// Store keys for user 1
	_, _ = storage.StoreKey(ctx, 1, ProviderOpenAI, "sk-openai-key-1234567890123456789012345")
	_, _ = storage.StoreKey(ctx, 1, ProviderAnthropic, "sk-ant-api03-key-123456789012345")
	_, _ = storage.StoreKey(ctx, 1, ProviderOllama, "ollama-key")

	// Store key for user 2
	_, _ = storage.StoreKey(ctx, 2, ProviderOpenAI, "sk-openai-key-9876543210987654321098765")

	// List keys for user 1
	user1Keys, err := storage.ListKeys(ctx, 1)
	if err != nil {
		t.Fatalf("ListKeys() error: %v", err)
	}
	if len(user1Keys) != 3 {
		t.Errorf("ListKeys() for user 1 returned %d keys, want 3", len(user1Keys))
	}

	// List keys for user 2
	user2Keys, err := storage.ListKeys(ctx, 2)
	if err != nil {
		t.Fatalf("ListKeys() error: %v", err)
	}
	if len(user2Keys) != 1 {
		t.Errorf("ListKeys() for user 2 returned %d keys, want 1", len(user2Keys))
	}

	// List keys for user with no keys
	user3Keys, err := storage.ListKeys(ctx, 3)
	if err != nil {
		t.Fatalf("ListKeys() error: %v", err)
	}
	if len(user3Keys) != 0 {
		t.Errorf("ListKeys() for user 3 returned %d keys, want 0", len(user3Keys))
	}
}

func TestKeyStorage_HasKey(t *testing.T) {
	storage, _ := NewInMemoryKeyStorage("test-master-key-12345")
	ctx := context.Background()

	// Initially no keys
	if storage.HasKey(ctx, 1, ProviderOpenAI) {
		t.Error("HasKey() should return false initially")
	}

	// Store a key
	_, _ = storage.StoreKey(ctx, 1, ProviderOpenAI, "sk-test-key-123456789012345678901234567890")

	// Now should exist
	if !storage.HasKey(ctx, 1, ProviderOpenAI) {
		t.Error("HasKey() should return true after store")
	}

	// Different user should not have the key
	if storage.HasKey(ctx, 2, ProviderOpenAI) {
		t.Error("HasKey() should return false for different user")
	}

	// Different provider should not have the key
	if storage.HasKey(ctx, 1, ProviderAnthropic) {
		t.Error("HasKey() should return false for different provider")
	}
}

func TestKeyStorage_MarkKeyUsed(t *testing.T) {
	storage, _ := NewInMemoryKeyStorage("test-master-key-12345")
	ctx := context.Background()

	// Store a key
	stored, _ := storage.StoreKey(ctx, 1, ProviderOpenAI, "sk-test-key-123456789012345678901234567890")

	// Initially LastUsedAt should be nil
	if stored.LastUsedAt != nil {
		t.Error("LastUsedAt should be nil initially")
	}

	// Mark as used
	err := storage.MarkKeyUsed(ctx, 1, ProviderOpenAI)
	if err != nil {
		t.Fatalf("MarkKeyUsed() error: %v", err)
	}

	// Get and verify LastUsedAt is set
	updated, _ := storage.GetStoredKey(ctx, 1, ProviderOpenAI)
	if updated.LastUsedAt == nil {
		t.Error("LastUsedAt should be set after MarkKeyUsed")
	}
}

func TestKeyStorage_MarkKeyUsed_NotFound(t *testing.T) {
	storage, _ := NewInMemoryKeyStorage("test-master-key-12345")
	ctx := context.Background()

	err := storage.MarkKeyUsed(ctx, 1, ProviderOpenAI)
	if err != ErrKeyNotFound {
		t.Errorf("MarkKeyUsed() error = %v, want ErrKeyNotFound", err)
	}
}

func TestKeyStorage_MultipleUsers(t *testing.T) {
	storage, _ := NewInMemoryKeyStorage("test-master-key-12345")
	ctx := context.Background()

	user1Key := "sk-user1-key-12345678901234567890123456"
	user2Key := "sk-user2-key-98765432109876543210987654"

	// Store keys for different users
	_, _ = storage.StoreKey(ctx, 1, ProviderOpenAI, user1Key)
	_, _ = storage.StoreKey(ctx, 2, ProviderOpenAI, user2Key)

	// Retrieve and verify they are separate
	retrieved1, _ := storage.GetKey(ctx, 1, ProviderOpenAI)
	retrieved2, _ := storage.GetKey(ctx, 2, ProviderOpenAI)

	if retrieved1 != user1Key {
		t.Errorf("User 1 key mismatch: got %v, want %v", retrieved1, user1Key)
	}
	if retrieved2 != user2Key {
		t.Errorf("User 2 key mismatch: got %v, want %v", retrieved2, user2Key)
	}

	// Delete user 1's key shouldn't affect user 2
	_ = storage.DeleteKey(ctx, 1, ProviderOpenAI)

	if storage.HasKey(ctx, 1, ProviderOpenAI) {
		t.Error("User 1 key should be deleted")
	}
	if !storage.HasKey(ctx, 2, ProviderOpenAI) {
		t.Error("User 2 key should still exist")
	}
}

func TestKeyStorage_InstanceLevelKey(t *testing.T) {
	storage, _ := NewInMemoryKeyStorage("test-master-key-12345")
	ctx := context.Background()

	// Use userID 0 for instance-level keys
	instanceKey := "sk-instance-level-key-1234567890123456"

	_, err := storage.StoreKey(ctx, 0, ProviderOpenAI, instanceKey)
	if err != nil {
		t.Fatalf("StoreKey() for instance key failed: %v", err)
	}

	retrieved, err := storage.GetKey(ctx, 0, ProviderOpenAI)
	if err != nil {
		t.Fatalf("GetKey() for instance key error: %v", err)
	}

	if retrieved != instanceKey {
		t.Errorf("Instance key mismatch: got %v, want %v", retrieved, instanceKey)
	}
}
