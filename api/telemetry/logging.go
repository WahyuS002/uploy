package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
)

var logFieldPattern = regexp.MustCompile(`(?:^|[\s,])([A-Za-z][A-Za-z0-9_-]*)=("[^"]*"|[^\s,]+)`)
var logSecretPattern = regexp.MustCompile(`(?i)(bearer\s+|(?:token|password|secret|private[_-]?key|authorization)=)(?:bearer\s+)?[^\s,]+`)
var logPEMPattern = regexp.MustCompile(`(?s)-----BEGIN [^-]+-----.*?-----END [^-]+-----`)

func init() {
	// JSON is consumable by log shippers while keeping the standard slog API.
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
}

// Printf preserves the small call-site API used by the application while
// emitting a structured record with common ID fields extracted as attributes.
func Printf(format string, args ...any) {
	write(fmt.Sprintf(format, args...))
}

// Println is the structured equivalent of log.Println.
func Println(args ...any) {
	write(strings.TrimSpace(fmt.Sprintln(args...)))
}

// Fatal logs an error and exits, matching the standard log package contract.
func Fatal(args ...any) {
	write(fmt.Sprint(args...))
	os.Exit(1)
}

// Fatalf logs a formatted error and exits, matching the standard log package contract.
func Fatalf(format string, args ...any) {
	write(fmt.Sprintf(format, args...))
	os.Exit(1)
}

func write(message string) {
	message = redact(message)
	level := slog.LevelInfo
	lower := strings.ToLower(message)
	if strings.Contains(lower, "panic") || strings.Contains(lower, "fatal") || strings.Contains(lower, "failed") || strings.Contains(lower, " error") || strings.HasSuffix(lower, "error") {
		level = slog.LevelError
	} else if strings.Contains(lower, "warning") || strings.Contains(lower, "degraded") {
		level = slog.LevelWarn
	}

	attrs := make([]slog.Attr, 0, 4)
	for _, match := range logFieldPattern.FindAllStringSubmatch(message, -1) {
		key := normalizeLogKey(match[1])
		value := strings.Trim(match[2], `"`)
		attrs = append(attrs, slog.String(key, value))
	}
	slog.LogAttrs(context.Background(), level, message, attrs...)
}

func normalizeLogKey(key string) string {
	for _, suffix := range []string{"ID", "Id"} {
		if strings.HasSuffix(key, suffix) && len(key) > len(suffix) {
			return strings.TrimSuffix(key, suffix) + "_id"
		}
	}
	return key
}

func redact(message string) string {
	message = logPEMPattern.ReplaceAllString(message, "[REDACTED]")
	return logSecretPattern.ReplaceAllString(message, "${1}[REDACTED]")
}
