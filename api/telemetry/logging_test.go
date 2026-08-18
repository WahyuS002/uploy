package telemetry

import (
	"strings"
	"testing"
)

func TestRedactRemovesSecrets(t *testing.T) {
	message := "token=super-secret authorization=Bearer abc123 -----BEGIN PRIVATE KEY-----secret-----END PRIVATE KEY-----"
	redacted := redact(message)
	for _, secret := range []string{"super-secret", "abc123", "PRIVATE KEY-----secret"} {
		if strings.Contains(redacted, secret) {
			t.Errorf("redacted message still contains %q: %s", secret, redacted)
		}
	}
}
