package llm

import (
	"strings"
	"testing"
)

func TestNewKeyCrypto(t *testing.T) {
	tests := []struct {
		name      string
		masterKey string
		wantErr   error
	}{
		{
			name:      "valid key",
			masterKey: "this-is-a-valid-master-key",
			wantErr:   nil,
		},
		{
			name:      "minimum length key",
			masterKey: "1234567890123456", // exactly 16 chars
			wantErr:   nil,
		},
		{
			name:      "key too short",
			masterKey: "short",
			wantErr:   ErrKeyTooShort,
		},
		{
			name:      "empty key",
			masterKey: "",
			wantErr:   ErrKeyTooShort,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			kc, err := NewKeyCrypto(tc.masterKey)
			if err != tc.wantErr {
				t.Errorf("NewKeyCrypto() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if tc.wantErr == nil && kc == nil {
				t.Error("NewKeyCrypto() returned nil without error")
			}
		})
	}
}

func TestKeyCrypto_EncryptDecrypt(t *testing.T) {
	kc, err := NewKeyCrypto("test-master-key-for-encryption")
	if err != nil {
		t.Fatalf("Failed to create KeyCrypto: %v", err)
	}

	tests := []struct {
		name      string
		plaintext string
	}{
		{
			name:      "simple API key",
			plaintext: "sk-test-api-key-12345",
		},
		{
			name:      "long API key",
			plaintext: "sk-ant-api03-very-long-anthropic-key-that-is-quite-long-indeed-12345678901234567890",
		},
		{
			name:      "empty string",
			plaintext: "",
		},
		{
			name:      "special characters",
			plaintext: "sk-!@#$%^&*()_+-=[]{}|;':\",./<>?",
		},
		{
			name:      "unicode characters",
			plaintext: "sk-测试-キー-مفتاح-🔐",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Encrypt
			ciphertext, err := kc.Encrypt(tc.plaintext)
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}

			// Empty plaintext should return empty ciphertext
			if tc.plaintext == "" {
				if ciphertext != "" {
					t.Error("Encrypt() should return empty string for empty plaintext")
				}
				return
			}

			// Ciphertext should be different from plaintext
			if ciphertext == tc.plaintext {
				t.Error("Encrypt() ciphertext should be different from plaintext")
			}

			// Decrypt
			decrypted, err := kc.Decrypt(ciphertext)
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}

			// Decrypted text should match original
			if decrypted != tc.plaintext {
				t.Errorf("Decrypt() = %v, want %v", decrypted, tc.plaintext)
			}
		})
	}
}

func TestKeyCrypto_EncryptProducesUniqueCiphertext(t *testing.T) {
	kc, err := NewKeyCrypto("test-master-key-for-encryption")
	if err != nil {
		t.Fatalf("Failed to create KeyCrypto: %v", err)
	}

	plaintext := "sk-test-api-key"

	// Encrypt the same plaintext multiple times
	ciphertexts := make([]string, 5)
	for i := 0; i < 5; i++ {
		ciphertext, err := kc.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("Encrypt() error = %v", err)
		}
		ciphertexts[i] = ciphertext
	}

	// All ciphertexts should be unique (due to random nonce)
	for i := 0; i < len(ciphertexts); i++ {
		for j := i + 1; j < len(ciphertexts); j++ {
			if ciphertexts[i] == ciphertexts[j] {
				t.Error("Encrypt() should produce unique ciphertexts for same plaintext")
			}
		}
	}

	// But all should decrypt to the same plaintext
	for i, ct := range ciphertexts {
		decrypted, err := kc.Decrypt(ct)
		if err != nil {
			t.Fatalf("Decrypt() error = %v", err)
		}
		if decrypted != plaintext {
			t.Errorf("Decrypt() of ciphertext[%d] = %v, want %v", i, decrypted, plaintext)
		}
	}
}

func TestKeyCrypto_DecryptWithWrongKey(t *testing.T) {
	kc1, _ := NewKeyCrypto("first-master-key-123")
	kc2, _ := NewKeyCrypto("second-master-key-456")

	plaintext := "sk-test-api-key"

	// Encrypt with first key
	ciphertext, err := kc1.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Try to decrypt with second key (should fail)
	_, err = kc2.Decrypt(ciphertext)
	if err == nil {
		t.Error("Decrypt() should fail with wrong key")
	}
}

func TestKeyCrypto_DecryptInvalidCiphertext(t *testing.T) {
	kc, _ := NewKeyCrypto("test-master-key-for-encryption")

	tests := []struct {
		name       string
		ciphertext string
	}{
		{
			name:       "invalid base64",
			ciphertext: "not-valid-base64!!!",
		},
		{
			name:       "too short",
			ciphertext: "YWJj", // "abc" in base64, too short for nonce
		},
		{
			name:       "corrupted ciphertext",
			ciphertext: "YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXo=", // valid base64 but not valid ciphertext
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := kc.Decrypt(tc.ciphertext)
			if err == nil {
				t.Error("Decrypt() should fail for invalid ciphertext")
			}
		})
	}
}

func TestKeyCrypto_DecryptEmpty(t *testing.T) {
	kc, _ := NewKeyCrypto("test-master-key-for-encryption")

	decrypted, err := kc.Decrypt("")
	if err != nil {
		t.Errorf("Decrypt() error = %v for empty string", err)
	}
	if decrypted != "" {
		t.Errorf("Decrypt() = %v, want empty string", decrypted)
	}
}

func TestMaskAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		apiKey   string
		expected string
	}{
		{
			name:     "standard OpenAI key",
			apiKey:   "sk-test1234567890abcdef",
			expected: "*******************cdef",
		},
		{
			name:     "short key (4 chars)",
			apiKey:   "1234",
			expected: "****",
		},
		{
			name:     "very short key (3 chars)",
			apiKey:   "123",
			expected: "***",
		},
		{
			name:     "5 char key",
			apiKey:   "12345",
			expected: "*2345",
		},
		{
			name:     "empty key",
			apiKey:   "",
			expected: "",
		},
		{
			name:     "long Anthropic key",
			apiKey:   "sk-ant-api03-1234567890abcdefghijklmnop",
			expected: "***********************************mnop",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := MaskAPIKey(tc.apiKey)
			if result != tc.expected {
				t.Errorf("MaskAPIKey(%q) = %q, want %q", tc.apiKey, result, tc.expected)
			}
		})
	}
}

func TestValidateAPIKeyFormat(t *testing.T) {
	tests := []struct {
		name         string
		providerType ProviderType
		apiKey       string
		wantErr      bool
		errContains  string
	}{
		// OpenAI tests
		{
			name:         "valid OpenAI key",
			providerType: ProviderOpenAI,
			apiKey:       "sk-1234567890abcdefghijklmnop",
			wantErr:      false,
		},
		{
			name:         "OpenAI key wrong prefix",
			providerType: ProviderOpenAI,
			apiKey:       "pk-1234567890abcdefghijklmnop",
			wantErr:      true,
			errContains:  "should start with 'sk-'",
		},
		{
			name:         "OpenAI key too short",
			providerType: ProviderOpenAI,
			apiKey:       "sk-short",
			wantErr:      true,
			errContains:  "too short",
		},

		// Anthropic tests
		{
			name:         "valid Anthropic key",
			providerType: ProviderAnthropic,
			apiKey:       "sk-ant-api03-1234567890",
			wantErr:      false,
		},
		{
			name:         "Anthropic key wrong prefix",
			providerType: ProviderAnthropic,
			apiKey:       "sk-1234567890",
			wantErr:      true,
			errContains:  "should start with 'sk-ant-'",
		},

		// Gemini tests
		{
			name:         "valid Gemini key",
			providerType: ProviderGemini,
			apiKey:       "AIzaSy1234567890abcdefghijk",
			wantErr:      false,
		},
		{
			name:         "Gemini key too short",
			providerType: ProviderGemini,
			apiKey:       "AIzaSy",
			wantErr:      true,
			errContains:  "too short",
		},

		// Ollama tests
		{
			name:         "Ollama empty key allowed",
			providerType: ProviderOllama,
			apiKey:       "",
			wantErr:      true, // Empty is caught by general check first
			errContains:  "cannot be empty",
		},
		{
			name:         "Ollama any key allowed",
			providerType: ProviderOllama,
			apiKey:       "any-key-works",
			wantErr:      false,
		},

		// General tests
		{
			name:         "empty key",
			providerType: ProviderOpenAI,
			apiKey:       "",
			wantErr:      true,
			errContains:  "cannot be empty",
		},
		{
			name:         "unknown provider valid key",
			providerType: "unknown",
			apiKey:       "some-valid-key-here",
			wantErr:      false,
		},
		{
			name:         "unknown provider short key",
			providerType: "unknown",
			apiKey:       "short",
			wantErr:      true,
			errContains:  "too short",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateAPIKeyFormat(tc.providerType, tc.apiKey)
			if tc.wantErr {
				if err == nil {
					t.Error("ValidateAPIKeyFormat() expected error, got nil")
				} else if tc.errContains != "" && !strings.Contains(err.Error(), tc.errContains) {
					t.Errorf("ValidateAPIKeyFormat() error = %v, want error containing %q", err, tc.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateAPIKeyFormat() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestGenerateKeyID(t *testing.T) {
	tests := []struct {
		name   string
		apiKey string
	}{
		{
			name:   "standard key",
			apiKey: "sk-test-api-key-12345",
		},
		{
			name:   "different key",
			apiKey: "sk-different-key-67890",
		},
		{
			name:   "empty key",
			apiKey: "",
		},
	}

	keyIDs := make(map[string]string)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			keyID := GenerateKeyID(tc.apiKey)

			// Empty key should return empty ID
			if tc.apiKey == "" {
				if keyID != "" {
					t.Errorf("GenerateKeyID(%q) = %q, want empty string", tc.apiKey, keyID)
				}
				return
			}

			// Key ID should be 8 hex characters
			if len(keyID) != 8 {
				t.Errorf("GenerateKeyID() length = %d, want 8", len(keyID))
			}

			// Key ID should be deterministic
			keyID2 := GenerateKeyID(tc.apiKey)
			if keyID != keyID2 {
				t.Error("GenerateKeyID() should be deterministic")
			}

			// Different keys should produce different IDs
			if existingKey, exists := keyIDs[keyID]; exists && existingKey != tc.apiKey {
				t.Errorf("GenerateKeyID() collision: %q and %q both produce %q", tc.apiKey, existingKey, keyID)
			}
			keyIDs[keyID] = tc.apiKey
		})
	}
}

func TestGenerateKeyID_Uniqueness(t *testing.T) {
	// Generate IDs for many different keys and check for collisions
	keys := []string{
		"sk-openai-key-1",
		"sk-openai-key-2",
		"sk-ant-api03-key-1",
		"sk-ant-api03-key-2",
		"AIzaSy-gemini-key-1",
		"AIzaSy-gemini-key-2",
	}

	ids := make(map[string]string)
	for _, key := range keys {
		id := GenerateKeyID(key)
		if existing, exists := ids[id]; exists {
			t.Errorf("Collision: keys %q and %q produce same ID %q", key, existing, id)
		}
		ids[id] = key
	}
}
