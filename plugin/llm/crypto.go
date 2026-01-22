package llm

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
)

var (
	// ErrInvalidCiphertext indicates the ciphertext is malformed.
	ErrInvalidCiphertext = errors.New("invalid ciphertext")

	// ErrKeyTooShort indicates the encryption key is too short.
	ErrKeyTooShort = errors.New("encryption key too short")
)

// KeyCrypto provides AES-256-GCM encryption for API keys.
// GCM (Galois/Counter Mode) provides both confidentiality and authenticity.
type KeyCrypto struct {
	key []byte
}

// NewKeyCrypto creates a new KeyCrypto instance with the given key.
// The key is hashed with SHA-256 to ensure it's exactly 32 bytes for AES-256.
func NewKeyCrypto(masterKey string) (*KeyCrypto, error) {
	if len(masterKey) < 16 {
		return nil, ErrKeyTooShort
	}

	// Hash the master key to get exactly 32 bytes for AES-256
	hash := sha256.Sum256([]byte(masterKey))
	return &KeyCrypto{key: hash[:]}, nil
}

// Encrypt encrypts plaintext using AES-256-GCM.
// Returns base64-encoded ciphertext (nonce || ciphertext || tag).
func (kc *KeyCrypto) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(kc.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Create a unique nonce for this encryption
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and prepend nonce
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Encode as base64 for safe storage
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts base64-encoded ciphertext using AES-256-GCM.
func (kc *KeyCrypto) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	block, err := aes.NewCipher(kc.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", ErrInvalidCiphertext
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// MaskAPIKey masks an API key for display, showing only the last 4 characters.
// Returns "****XXXX" format where XXXX is the last 4 chars.
func MaskAPIKey(apiKey string) string {
	if apiKey == "" {
		return ""
	}

	if len(apiKey) <= 4 {
		return strings.Repeat("*", len(apiKey))
	}

	lastFour := apiKey[len(apiKey)-4:]
	return strings.Repeat("*", len(apiKey)-4) + lastFour
}

// ValidateAPIKeyFormat performs basic format validation for API keys.
// Different providers have different key formats.
func ValidateAPIKeyFormat(providerType ProviderType, apiKey string) error {
	if apiKey == "" {
		return errors.New("API key cannot be empty")
	}

	switch providerType {
	case ProviderOpenAI:
		// OpenAI keys typically start with "sk-" and are ~51 chars
		if !strings.HasPrefix(apiKey, "sk-") {
			return errors.New("OpenAI API key should start with 'sk-'")
		}
		if len(apiKey) < 20 {
			return errors.New("OpenAI API key appears too short")
		}
	case ProviderAnthropic:
		// Anthropic keys typically start with "sk-ant-"
		if !strings.HasPrefix(apiKey, "sk-ant-") {
			return errors.New("Anthropic API key should start with 'sk-ant-'")
		}
	case ProviderGemini:
		// Google API keys are typically 39 characters
		if len(apiKey) < 20 {
			return errors.New("Google API key appears too short")
		}
	case ProviderOllama:
		// Ollama doesn't require an API key
		return nil
	default:
		// For unknown providers, just check non-empty
		if len(apiKey) < 10 {
			return errors.New("API key appears too short")
		}
	}

	return nil
}

// GenerateKeyID generates a unique identifier for an API key.
// This uses the first 8 chars of the SHA-256 hash of the key.
func GenerateKeyID(apiKey string) string {
	if apiKey == "" {
		return ""
	}
	hash := sha256.Sum256([]byte(apiKey))
	return fmt.Sprintf("%x", hash[:4])
}
