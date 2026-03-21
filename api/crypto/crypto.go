package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
)

var (
	gcm               cipher.AEAD
	errNotInitialized = fmt.Errorf("crypto: not initialized, call crypto.Init() first")
)

// Init derives an AES-256-GCM cipher from a 64-char hex-encoded key.
func Init(hexKey string) error {
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return fmt.Errorf("crypto: invalid hex key: %w", err)
	}
	if len(key) != 32 {
		return fmt.Errorf("crypto: key must be 32 bytes (64 hex chars), got %d bytes", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("crypto: %w", err)
	}
	gcm, err = cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("crypto: %w", err)
	}
	return nil
}

// Encrypt encrypts plaintext using AES-256-GCM and returns a base64-encoded string.
func Encrypt(plaintext string) (string, error) {
	if gcm == nil {
		return "", errNotInitialized
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("crypto: failed to generate nonce: %w", err)
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a base64-encoded AES-256-GCM ciphertext back to plaintext.
func Decrypt(encoded string) (string, error) {
	if gcm == nil {
		return "", errNotInitialized
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("crypto: invalid base64: %w", err)
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("crypto: ciphertext too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("crypto: decryption failed: %w", err)
	}
	return string(plaintext), nil
}
