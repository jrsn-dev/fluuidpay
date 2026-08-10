package platform

import (
	"log/slog"
	"os"
	"strings"
)

// NewLogger creates a structured logger based on the environment.
// In production (LOG_FORMAT=json), uses JSON output.
// In development, uses human-readable text output.
func NewLogger() *slog.Logger {
	format := strings.ToLower(os.Getenv("LOG_FORMAT"))
	level := parseLogLevel(os.Getenv("LOG_LEVEL"))

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: level == slog.LevelDebug,
	}

	var handler slog.Handler
	if format == "json" || format == "" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}

func parseLogLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// RedactSensitiveFields returns a set of field names that must be redacted
// from logs to comply with PCI-DSS and privacy requirements.
func RedactSensitiveFields() map[string]bool {
	return map[string]bool{
		"card_token":    true,
		"card_number":   true,
		"pan":           true,
		"cvv":           true,
		"cvc":           true,
		"password":      true,
		"secret":        true,
		"api_key":       true,
		"authorization": true,
	}
}

// SanitizeForLog removes or masks sensitive values from a map before logging.
func SanitizeForLog(data map[string]any) map[string]any {
	redact := RedactSensitiveFields()
	result := make(map[string]any, len(data))
	for k, v := range data {
		if redact[strings.ToLower(k)] {
			result[k] = "***REDACTED***"
		} else {
			result[k] = v
		}
	}
	return result
}
