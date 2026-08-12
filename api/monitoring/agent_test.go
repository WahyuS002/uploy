package monitoring

import "testing"

func TestPinnedImage(t *testing.T) {
	for _, image := range []string{"ghcr.io/acme/monitor:v1.2.3", "ghcr.io/acme/monitor@sha256:abc"} {
		if !pinnedImage(image) {
			t.Fatalf("expected pinned image: %s", image)
		}
	}
	for _, image := range []string{"ghcr.io/acme/monitor", "ghcr.io/acme/monitor:latest"} {
		if pinnedImage(image) {
			t.Fatalf("expected unpinned image: %s", image)
		}
	}
}

func TestPrivateURL(t *testing.T) {
	if got := PrivateURL("10.0.0.4", 9184); got != "http://10.0.0.4:9184" {
		t.Fatalf("PrivateURL = %q", got)
	}
}

func TestPrivateIP(t *testing.T) {
	for _, value := range []string{"10.0.0.4", "192.168.1.4", "100.64.0.2", "fd00::2"} {
		if !privateIP(value) {
			t.Fatalf("expected private IP: %s", value)
		}
	}
	for _, value := range []string{"127.0.0.1", "0.0.0.0", "8.8.8.8", "example.com"} {
		if privateIP(value) {
			t.Fatalf("expected rejected IP: %s", value)
		}
	}
}

func TestValidateConfigRequiresBothTokens(t *testing.T) {
	config := Config{Image: "ghcr.io/acme/monitor:v1", PrivateAddress: "10.0.0.4", HostPort: 9184, RetentionDays: 7, ControlToken: "control"}
	if err := ValidateConfig(config); err == nil {
		t.Fatal("expected missing reader token error")
	}
}
