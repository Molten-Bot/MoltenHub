package model

import (
	"encoding/json"
	"fmt"
	"strings"
)

// NormalizeAgentSkillName applies the canonical advertised skill name rules.
func NormalizeAgentSkillName(raw string) (string, bool) {
	return normalizeSkillToken(raw, 2)
}

// NormalizeAgentSkillParameterName applies the canonical skill parameter name rules.
func NormalizeAgentSkillParameterName(raw string) (string, bool) {
	return normalizeSkillToken(raw, 1)
}

func normalizeSkillToken(raw string, minLen int) (string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if len(normalized) < minLen || len(normalized) > 64 {
		return "", false
	}
	for _, ch := range normalized {
		switch {
		case ch >= 'a' && ch <= 'z':
		case ch >= '0' && ch <= '9':
		case ch == '-', ch == '_', ch == '.':
		default:
			return "", false
		}
	}
	return normalized, true
}

// ValidateSkillParameterPayloadKeys checks required and unknown skill payload keys.
func ValidateSkillParameterPayloadKeys(provided map[string]string, required []string, allowed map[string]struct{}) []string {
	errors := []string{}
	for _, name := range required {
		if strings.TrimSpace(provided[name]) == "" {
			errors = append(errors, fmt.Sprintf("missing required parameter %q", name))
		}
	}
	for name := range provided {
		if _, ok := allowed[name]; !ok {
			errors = append(errors, fmt.Sprintf("unknown parameter %q", name))
		}
	}
	return errors
}

// QuoteString returns a JSON-compatible quoted string for API error details.
func QuoteString(value string) string {
	body, _ := json.Marshal(value)
	return string(body)
}
