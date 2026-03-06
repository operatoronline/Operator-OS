package auth

import (
	"bytes"
	"testing"
)

func TestEncryptDecryptRoundtrip(t *testing.T) {
	plaintext := []byte("hello, world! this is a secret message.")
	passphrase := "my-strong-passphrase"

	encrypted, err := encryptAESGCM(plaintext, passphrase)
	if err != nil {
		t.Fatalf("encryptAESGCM() error: %v", err)
	}

	if bytes.Equal(encrypted, plaintext) {
		t.Error("encrypted data should differ from plaintext")
	}

	decrypted, err := decryptAESGCM(encrypted, passphrase)
	if err != nil {
		t.Fatalf("decryptAESGCM() error: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("decrypted = %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptDifferentCiphertexts(t *testing.T) {
	// Same plaintext should produce different ciphertexts (random salt/nonce).
	plaintext := []byte("same input")
	passphrase := "test"

	enc1, err := encryptAESGCM(plaintext, passphrase)
	if err != nil {
		t.Fatalf("encryptAESGCM() error: %v", err)
	}
	enc2, err := encryptAESGCM(plaintext, passphrase)
	if err != nil {
		t.Fatalf("encryptAESGCM() error: %v", err)
	}

	if bytes.Equal(enc1, enc2) {
		t.Error("two encryptions of same plaintext should differ (random salt/nonce)")
	}
}

func TestDecryptWrongPassphrase(t *testing.T) {
	plaintext := []byte("secret data")
	encrypted, err := encryptAESGCM(plaintext, "correct-key")
	if err != nil {
		t.Fatalf("encryptAESGCM() error: %v", err)
	}

	_, err = decryptAESGCM(encrypted, "wrong-key")
	if err == nil {
		t.Error("expected error when decrypting with wrong passphrase")
	}
}

func TestDecryptTooShort(t *testing.T) {
	_, err := decryptAESGCM([]byte("short"), "key")
	if err == nil {
		t.Error("expected error for too-short ciphertext")
	}
}

func TestDeriveKeyDeterministic(t *testing.T) {
	salt := []byte("0123456789abcdef")
	key1 := deriveKey("passphrase", salt)
	key2 := deriveKey("passphrase", salt)

	if !bytes.Equal(key1, key2) {
		t.Error("deriveKey should be deterministic for same passphrase and salt")
	}

	if len(key1) != argonKeyLen {
		t.Errorf("key length = %d, want %d", len(key1), argonKeyLen)
	}
}

func TestDeriveKeyDifferentSalts(t *testing.T) {
	salt1 := []byte("0123456789abcdef")
	salt2 := []byte("fedcba9876543210")

	key1 := deriveKey("passphrase", salt1)
	key2 := deriveKey("passphrase", salt2)

	if bytes.Equal(key1, key2) {
		t.Error("different salts should produce different keys")
	}
}

func TestEncryptEmptyPlaintext(t *testing.T) {
	encrypted, err := encryptAESGCM([]byte{}, "key")
	if err != nil {
		t.Fatalf("encryptAESGCM() error: %v", err)
	}

	decrypted, err := decryptAESGCM(encrypted, "key")
	if err != nil {
		t.Fatalf("decryptAESGCM() error: %v", err)
	}

	if len(decrypted) != 0 {
		t.Errorf("expected empty decrypted data, got %d bytes", len(decrypted))
	}
}
