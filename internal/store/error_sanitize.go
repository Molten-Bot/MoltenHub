package store

import (
	"context"
	"errors"
	"strings"
)

func SanitizeError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.Canceled) {
		return "request canceled"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "request timed out"
	}
	return SanitizeErrorText(err.Error())
}

func SanitizeErrorText(text string) string {
	msg := strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
	if msg == "" {
		return ""
	}

	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "context canceled") || strings.Contains(lower, "request canceled"):
		return "request canceled"
	case strings.Contains(lower, "deadline exceeded") || strings.Contains(lower, "timed out") || strings.Contains(lower, "timeout"):
		return "request timed out"
	case strings.Contains(lower, "status 401") || strings.Contains(lower, "status 403") || strings.Contains(lower, "authorization") || strings.Contains(lower, "signature"):
		return "authorization failed"
	case strings.Contains(lower, "connection refused") || strings.Contains(lower, "no such host") || strings.Contains(lower, "network is unreachable") || strings.Contains(lower, "dial tcp") || strings.Contains(lower, "connection reset"):
		return "connection failed"
	case strings.Contains(lower, "status 404"):
		return "resource not found"
	case strings.Contains(lower, "status ") || strings.Contains(msg, "\n") || strings.Contains(msg, "http://") || strings.Contains(msg, "https://") || strings.Contains(msg, "<") || len(msg) > 160:
		return "request failed"
	default:
		return msg
	}
}
