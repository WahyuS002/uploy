package ssh

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"

	gossh "golang.org/x/crypto/ssh"
)

// DerivePublicKey parses an OpenSSH private key PEM and returns the
// corresponding public key in OpenSSH authorized_keys one-line format.
func DerivePublicKey(privateKeyPEM string) (string, error) {
	signer, err := gossh.ParsePrivateKey([]byte(privateKeyPEM))
	if err != nil {
		return "", fmt.Errorf("parse private key: %w", err)
	}
	return string(gossh.MarshalAuthorizedKey(signer.PublicKey())), nil
}

// GenerateEd25519Key creates a new Ed25519 keypair in memory and returns
// the private key as an OpenSSH PEM string.
func GenerateEd25519Key() (string, error) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", fmt.Errorf("generate ed25519 key: %w", err)
	}

	pemBlock, err := gossh.MarshalPrivateKey(priv, "")
	if err != nil {
		return "", fmt.Errorf("marshal private key: %w", err)
	}

	return string(pem.EncodeToMemory(pemBlock)), nil
}
