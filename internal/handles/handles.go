package handles

import (
	"errors"
	"regexp"
	"strings"
)

const (
	MinLength = 2
	MaxLength = 64
)

var (
	ErrInvalidHandle   = errors.New("invalid handle")
	ErrBlockedHandle   = errors.New("blocked handle")
	ErrInvalidAgentRef = errors.New("invalid agent reference")

	handleRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{1,63}$`)
	blockedSet  = map[string]struct{}{
		"asshole":      {},
		"bastard":      {},
		"bitch":        {},
		"bullshit":     {},
		"cunt":         {},
		"dick":         {},
		"fuck":         {},
		"fucker":       {},
		"fucking":      {},
		"motherfucker": {},
		"nigger":       {},
		"nigga":        {},
		"pussy":        {},
		"shit":         {},
		"shitty":       {},
		"slut":         {},
		"whore":        {},
	}
)

func Normalize(raw string) string {
	in := strings.TrimSpace(strings.ToLower(raw))
	if in == "" {
		return ""
	}
	var b strings.Builder
	prevSep := false
	for _, r := range in {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevSep = false
			continue
		}
		switch r {
		case '-', '_', '.':
			if b.Len() == 0 || prevSep {
				continue
			}
			b.WriteRune(r)
			prevSep = true
		default:
			if b.Len() == 0 || prevSep {
				continue
			}
			b.WriteRune('-')
			prevSep = true
		}
	}
	out := strings.Trim(b.String(), "._-")
	if len(out) > MaxLength {
		out = strings.Trim(out[:MaxLength], "._-")
	}
	return out
}

func NormalizeAgentRef(raw string) string {
	trimmed := strings.TrimSpace(strings.ToLower(raw))
	if trimmed == "" {
		return ""
	}
	if !strings.Contains(trimmed, "/") {
		return Normalize(trimmed)
	}
	parts := strings.Split(strings.Trim(trimmed, "/"), "/")
	normalized := make([]string, 0, len(parts))
	for _, part := range parts {
		h := Normalize(part)
		if h == "" {
			continue
		}
		normalized = append(normalized, h)
	}
	return strings.Join(normalized, "/")
}

func ValidateHandle(handle string) error {
	if !handleRegex.MatchString(handle) {
		return ErrInvalidHandle
	}
	if IsBlocked(handle) {
		return ErrBlockedHandle
	}
	return nil
}

func ValidateAgentRef(ref string) error {
	if strings.Contains(ref, "/") {
		parts := strings.Split(strings.Trim(ref, "/"), "/")
		if len(parts) != 2 && len(parts) != 3 && len(parts) != 4 {
			return ErrInvalidAgentRef
		}
		for _, p := range parts {
			if err := ValidateHandle(p); err != nil {
				return ErrInvalidAgentRef
			}
		}
		return nil
	}
	if err := ValidateHandle(ref); err != nil {
		return ErrInvalidAgentRef
	}
	return nil
}

func BuildAgentURI(orgHandle string, ownerHumanHandle *string, agentHandle string) string {
	org := Normalize(orgHandle)
	agent := Normalize(agentHandle)
	if ownerHumanHandle == nil || strings.TrimSpace(*ownerHumanHandle) == "" {
		return org + "/" + agent
	}
	human := Normalize(*ownerHumanHandle)
	return org + "/" + human + "/" + agent
}

func BuildHumanAgentURI(ownerHumanHandle string, agentHandle string) string {
	human := Normalize(ownerHumanHandle)
	agent := Normalize(agentHandle)
	return "human/" + human + "/agent/" + agent
}

func IsBlocked(handle string) bool {
	in := strings.ToLower(strings.TrimSpace(handle))
	if in == "" {
		return false
	}

	tokens := splitAlphaNumTokens(in)
	for _, token := range tokens {
		if _, blocked := blockedSet[token]; blocked {
			return true
		}
	}

	compact := compactAlphaNum(in)
	if compact == "" {
		return false
	}
	if _, blocked := blockedSet[compact]; blocked {
		return true
	}
	return false
}

func splitAlphaNumTokens(in string) []string {
	parts := strings.FieldsFunc(in, func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'))
	})
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

func compactAlphaNum(in string) string {
	var b strings.Builder
	for _, r := range in {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}
