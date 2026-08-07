package db

import (
	"testing"

	"github.com/WahyuS002/uploy/crypto"
)

// Env var values are encrypted at rest, but rows written before that change are
// still plaintext. Reads have to handle both without mangling either.
func TestDecryptEnvValue(t *testing.T) {
	if err := crypto.Init("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"); err != nil {
		t.Fatalf("crypto.Init: %v", err)
	}

	const secret = "postgres://user:pa55w0rd@host/db"

	encrypted, err := crypto.Encrypt(secret)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if encrypted == secret {
		t.Fatal("value was stored unencrypted")
	}
	if got := decryptEnvValue("svc-1", "DATABASE_URL", encrypted); got != secret {
		t.Errorf("encrypted round-trip = %q, want %q", got, secret)
	}

	// A legacy plaintext row fails to decrypt and must come back untouched.
	if got := decryptEnvValue("svc-1", "DATABASE_URL", secret); got != secret {
		t.Errorf("legacy plaintext = %q, want %q", got, secret)
	}
}
