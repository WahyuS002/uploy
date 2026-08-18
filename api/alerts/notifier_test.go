package alerts

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSendDiscord(t *testing.T) {
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		data, _ := io.ReadAll(r.Body)
		body = string(data)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	err := Send(context.Background(), Channel{Type: "discord", Enabled: true, Config: map[string]interface{}{"url": server.URL}}, Message{Title: "[uploy] CPU high", Body: "api is above 85%", StartedAt: time.Now()})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if !strings.Contains(body, "CPU high") || !strings.Contains(body, "api is above") {
		t.Fatalf("payload = %s", body)
	}
}

func TestValidateChannelConfig(t *testing.T) {
	if err := ValidateChannelConfig("telegram", map[string]interface{}{"bot_token": "token"}); err == nil {
		t.Fatal("telegram without chat ID should be rejected")
	}
	if err := ValidateChannelConfig("webhook", map[string]interface{}{"url": "https://example.test/hook"}); err != nil {
		t.Fatalf("valid webhook rejected: %v", err)
	}
}
