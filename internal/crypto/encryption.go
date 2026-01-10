package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
)

// Encryptor handles encryption and decryption of sensitive data
type Encryptor struct {
	key []byte
}

// NewEncryptor creates a new encryptor with the given key
// Key should be 32 bytes for AES-256
func NewEncryptor(key []byte) (*Encryptor, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes, got %d", len(key))
	}
	return &Encryptor{key: key}, nil
}

// NewEncryptorFromEnv creates a new encryptor from ENCRYPTION_KEY environment variable
func NewEncryptorFromEnv() (*Encryptor, error) {
	keyStr := os.Getenv("ENCRYPTION_KEY")
	if keyStr == "" {
		return nil, fmt.Errorf("ENCRYPTION_KEY environment variable not set")
	}

	// Decode base64-encoded key
	key, err := base64.StdEncoding.DecodeString(keyStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ENCRYPTION_KEY: %w", err)
	}

	return NewEncryptor(key)
}

// Encrypt encrypts plaintext using AES-256-GCM
func (e *Encryptor) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts ciphertext encrypted with Encrypt
func (e *Encryptor) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	ct, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		// If decoding fails, assume it's not encrypted (for backwards compatibility)
		return ciphertext, nil
	}

	nonceSize := aead.NonceSize()
	if len(ct) < nonceSize {
		// Invalid ciphertext, return as-is
		return ciphertext, nil
	}

	nonce, ct := ct[:nonceSize], ct[nonceSize:]
	plaintext, err := aead.Open(nil, nonce, ct, nil)
	if err != nil {
		// If decryption fails, assume it's not encrypted (for backwards compatibility)
		return ciphertext, nil
	}

	return string(plaintext), nil
}

// EncryptIfNeeded encrypts plaintext only if it's not already masked
func (e *Encryptor) EncryptIfNeeded(plaintext string, maskedValue string) (string, error) {
	if plaintext == "" || plaintext == maskedValue {
		return plaintext, nil
	}
	return e.Encrypt(plaintext)
}

// IsMasked checks if a value is a masked value (should not be updated)
func IsMasked(value string) bool {
	return value == "••••••••••••••••••••" || value == ""
}
