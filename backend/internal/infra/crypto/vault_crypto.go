package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
)

// VaultEncryptor encrypts and decrypts vault secret values using AES-256-GCM.
type VaultEncryptor struct {
	aead cipher.AEAD
}

// NewVaultEncryptor creates an encryptor from a 64-character hex-encoded key (32 bytes).
func NewVaultEncryptor(hexKey string) (*VaultEncryptor, error) {
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("decode encryption key: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes (64 hex chars), got %d bytes", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	return &VaultEncryptor{aead: aead}, nil
}

// Encrypt encrypts plaintext and returns (ciphertext, nonce).
// The aad (Additional Authenticated Data) binds the ciphertext to its storage
// context (e.g. vaultID + "|" + key) so it cannot be moved across rows.
func (e *VaultEncryptor) Encrypt(plaintext, aad []byte) (ciphertext, nonce []byte, err error) {
	nonce = make([]byte, e.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, fmt.Errorf("generate nonce: %w", err)
	}
	ciphertext = e.aead.Seal(nil, nonce, plaintext, aad)
	return ciphertext, nonce, nil
}

// Decrypt decrypts ciphertext using the given nonce and aad.
// The aad must match what was used during encryption.
func (e *VaultEncryptor) Decrypt(ciphertext, nonce, aad []byte) ([]byte, error) {
	plaintext, err := e.aead.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	return plaintext, nil
}

// SecretAAD builds the Additional Authenticated Data for a vault secret.
func SecretAAD(vaultID, key string) []byte {
	return []byte(vaultID + "|" + key)
}
