package crypto

import (
	"crypto/rand"
	"encoding/hex"
	"testing"
)

func newTestEncryptor(t *testing.T) *VaultEncryptor {
	t.Helper()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	enc, err := NewVaultEncryptor(hex.EncodeToString(key))
	if err != nil {
		t.Fatal(err)
	}
	return enc
}

func TestVaultEncryptorRoundTrip(t *testing.T) {
	enc := newTestEncryptor(t)
	aad := SecretAAD("vault_abc", "API_KEY")

	plaintext := []byte("my-secret-api-key-12345")
	ciphertext, nonce, err := enc.Encrypt(plaintext, aad)
	if err != nil {
		t.Fatal(err)
	}

	if string(ciphertext) == string(plaintext) {
		t.Fatal("ciphertext should differ from plaintext")
	}

	decrypted, err := enc.Decrypt(ciphertext, nonce, aad)
	if err != nil {
		t.Fatal(err)
	}

	if string(decrypted) != string(plaintext) {
		t.Fatalf("expected %q, got %q", plaintext, decrypted)
	}
}

func TestVaultEncryptorUniqueCiphertexts(t *testing.T) {
	enc := newTestEncryptor(t)
	aad := SecretAAD("vault_abc", "API_KEY")

	plaintext := []byte("same-value")
	c1, _, err := enc.Encrypt(plaintext, aad)
	if err != nil {
		t.Fatal(err)
	}
	c2, _, err := enc.Encrypt(plaintext, aad)
	if err != nil {
		t.Fatal(err)
	}

	if string(c1) == string(c2) {
		t.Fatal("encrypting the same plaintext twice should produce different ciphertexts")
	}
}

func TestVaultEncryptorBadKey(t *testing.T) {
	if _, err := NewVaultEncryptor("not-hex"); err == nil {
		t.Fatal("expected error for non-hex key")
	}
	if _, err := NewVaultEncryptor("aabb"); err == nil {
		t.Fatal("expected error for short key")
	}
}

func TestVaultEncryptorWrongNonce(t *testing.T) {
	enc := newTestEncryptor(t)
	aad := SecretAAD("vault_abc", "API_KEY")

	ciphertext, _, err := enc.Encrypt([]byte("secret"), aad)
	if err != nil {
		t.Fatal(err)
	}

	wrongNonce := make([]byte, 12)
	if _, err := rand.Read(wrongNonce); err != nil {
		t.Fatal(err)
	}

	if _, err := enc.Decrypt(ciphertext, wrongNonce, aad); err == nil {
		t.Fatal("expected decryption error with wrong nonce")
	}
}

func TestVaultEncryptorWrongAAD(t *testing.T) {
	enc := newTestEncryptor(t)

	aad1 := SecretAAD("vault_abc", "KEY_A")
	aad2 := SecretAAD("vault_xyz", "KEY_B")

	ciphertext, nonce, err := enc.Encrypt([]byte("secret"), aad1)
	if err != nil {
		t.Fatal(err)
	}

	// Decrypting with different AAD must fail — prevents ciphertext relocation attacks
	if _, err := enc.Decrypt(ciphertext, nonce, aad2); err == nil {
		t.Fatal("expected decryption error with wrong AAD")
	}
}
