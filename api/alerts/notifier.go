package alerts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

type Channel struct {
	ID      string
	Name    string
	Type    string
	Enabled bool
	Config  map[string]interface{}
}

type Message struct {
	Title      string
	Body       string
	Resolved   bool
	StartedAt  time.Time
	ResolvedAt time.Time
}

var httpClient = &http.Client{Timeout: 10 * time.Second}

func ValidateChannelConfig(kind string, config map[string]interface{}) error {
	switch kind {
	case "discord", "slack", "webhook":
		if requiredString(config, "url") == "" {
			return fmt.Errorf("%s channel requires url", kind)
		}
	case "telegram":
		if requiredString(config, "bot_token") == "" || requiredString(config, "chat_id") == "" {
			return fmt.Errorf("telegram channel requires bot_token and chat_id")
		}
	case "email":
		if requiredString(config, "smtp_host") == "" || requiredString(config, "from") == "" || requiredString(config, "to") == "" {
			return fmt.Errorf("email channel requires smtp_host, from, and to")
		}
		port := optionalInt(config, "smtp_port", 587)
		if port < 1 || port > 65535 {
			return fmt.Errorf("email smtp_port is invalid")
		}
	default:
		return fmt.Errorf("unsupported notification channel type %q", kind)
	}
	return nil
}

func Send(ctx context.Context, channel Channel, message Message) error {
	if !channel.Enabled {
		return nil
	}
	if err := ValidateChannelConfig(channel.Type, channel.Config); err != nil {
		return err
	}
	content := message.Body
	if message.Title != "" {
		content = message.Title + "\n" + content
	}
	switch channel.Type {
	case "discord":
		return sendJSON(ctx, requiredString(channel.Config, "url"), map[string]string{"content": content})
	case "slack":
		return sendJSON(ctx, requiredString(channel.Config, "url"), map[string]string{"text": content})
	case "webhook":
		return sendJSON(ctx, requiredString(channel.Config, "url"), map[string]interface{}{
			"title": message.Title, "message": message.Body, "resolved": message.Resolved,
			"started_at":  message.StartedAt.UTC().Format(time.RFC3339),
			"resolved_at": formatOptionalTime(message.ResolvedAt),
		})
	case "telegram":
		baseURL := requiredString(channel.Config, "api_url")
		if baseURL == "" {
			baseURL = "https://api.telegram.org"
		}
		url := strings.TrimRight(baseURL, "/") + "/bot" + requiredString(channel.Config, "bot_token") + "/sendMessage"
		return sendJSON(ctx, url, map[string]string{"chat_id": requiredString(channel.Config, "chat_id"), "text": content})
	case "email":
		return sendEmail(ctx, channel.Config, message.Title, content)
	default:
		return fmt.Errorf("unsupported notification channel type %q", channel.Type)
	}
}

func sendJSON(ctx context.Context, url string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("notification endpoint returned HTTP %d", response.StatusCode)
	}
	return nil
}

func sendEmail(ctx context.Context, config map[string]interface{}, subject, body string) error {
	host := requiredString(config, "smtp_host")
	port := optionalInt(config, "smtp_port", 587)
	from := requiredString(config, "from")
	to := requiredString(config, "to")
	username := requiredString(config, "username")
	password := requiredString(config, "password")
	message := "From: " + from + "\r\nTo: " + to + "\r\nSubject: " + subject + "\r\n\r\n" + body
	result := make(chan error, 1)
	go func() {
		var auth smtp.Auth
		if username != "" {
			auth = smtp.PlainAuth("", username, password, host)
		}
		result <- smtp.SendMail(host+":"+strconv.Itoa(port), auth, from, []string{to}, []byte(message))
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-result:
		return err
	}
}

func requiredString(config map[string]interface{}, key string) string {
	value, _ := config[key].(string)
	return strings.TrimSpace(value)
}

func optionalInt(config map[string]interface{}, key string, fallback int) int {
	switch value := config[key].(type) {
	case float64:
		return int(value)
	case int:
		return value
	case string:
		parsed, err := strconv.Atoi(value)
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func formatOptionalTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
