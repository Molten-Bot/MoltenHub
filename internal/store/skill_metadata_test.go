package store

import (
	"errors"
	"reflect"
	"testing"
)

func TestSecretMarkerHelpers(t *testing.T) {
	if !containsSecretProhibition("Do not include secrets in parameters") {
		t.Fatal("expected prohibition marker to be detected")
	}
	if containsSecretProhibition("Send any values") {
		t.Fatal("did not expect prohibition marker")
	}
	if !containsStrongSecretMarker("Authorization: Bearer abc") {
		t.Fatal("expected strong secret marker")
	}
	if !containsGenericSecretMarker("secret value") {
		t.Fatal("expected generic secret marker")
	}
	if !containsLikelySecret("use API key here") {
		t.Fatal("expected likely secret")
	}
	if containsLikelySecret("Never include secrets in the payload") {
		t.Fatal("did not expect prohibition text to be treated as secret leak")
	}
}

func TestValidateSkillParameterPayloadKeys(t *testing.T) {
	provided := map[string]string{
		"required": "",
		"unknown":  "x",
		"optional": "y",
	}
	errs := validateSkillParameterPayloadKeys(provided, []string{"required"}, map[string]struct{}{
		"required": {},
		"optional": {},
	})
	want := []string{
		`missing required parameter "required"`,
		`unknown parameter "unknown"`,
	}
	if !reflect.DeepEqual(errs, want) {
		t.Fatalf("unexpected validation errors: got=%v want=%v", errs, want)
	}
}

func TestMarkdownHelpers(t *testing.T) {
	if got := markdownParameterSection("## Required Parameters:"); got != "required" {
		t.Fatalf("unexpected markdown section: %q", got)
	}
	if got := markdownParameterSection("Optional:"); got != "optional" {
		t.Fatalf("unexpected markdown section: %q", got)
	}
	if got := firstNonEmpty("", "  ", "x", "y"); got != "x" {
		t.Fatalf("unexpected first non-empty value: %q", got)
	}
}

func TestNormalizeAgentSkillParameterName(t *testing.T) {
	if got, ok := normalizeAgentSkillParameterName(" Param.Name_1 "); !ok || got != "param.name_1" {
		t.Fatalf("expected normalized parameter name, got=%q ok=%v", got, ok)
	}
	if _, ok := normalizeAgentSkillParameterName("bad name with spaces"); ok {
		t.Fatal("expected invalid parameter name")
	}
}

func TestParseMarkdownSkillParameters(t *testing.T) {
	markdown := `Required:
- query: Search query string
Optional:
- timeout_ms: Timeout in milliseconds`

	required, optional, err := parseMarkdownSkillParameters(markdown)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if len(required) != 1 || len(optional) != 1 {
		t.Fatalf("unexpected parsed sizes required=%d optional=%d", len(required), len(optional))
	}
	if required[0]["name"] != "query" {
		t.Fatalf("unexpected required param name: %v", required[0]["name"])
	}
	if optional[0]["name"] != "timeout_ms" {
		t.Fatalf("unexpected optional param name: %v", optional[0]["name"])
	}

	_, _, err = parseMarkdownSkillParameters(`- query: no section`)
	if !errors.Is(err, ErrInvalidAgentSkills) {
		t.Fatalf("expected ErrInvalidAgentSkills for missing section, got %v", err)
	}

	_, _, err = parseMarkdownSkillParameters(`Required:
- query: first
- query: duplicate`)
	if !errors.Is(err, ErrInvalidAgentSkills) {
		t.Fatalf("expected ErrInvalidAgentSkills for duplicate names, got %v", err)
	}

	_, _, err = parseMarkdownSkillParameters(`Required:
- query: include API key here`)
	if !errors.Is(err, ErrInvalidSkillDescription) {
		t.Fatalf("expected ErrInvalidSkillDescription for secret-like description, got %v", err)
	}
}

func TestNormalizeMarkdownSkillParameters(t *testing.T) {
	_, err := normalizeMarkdownSkillParameters("")
	if !errors.Is(err, ErrInvalidAgentSkills) {
		t.Fatalf("expected ErrInvalidAgentSkills for empty markdown, got %v", err)
	}

	_, err = normalizeMarkdownSkillParameters(`Required:
- query: Search query`)
	if !errors.Is(err, ErrInvalidSkillDescription) {
		t.Fatalf("expected ErrInvalidSkillDescription when prohibition text missing, got %v", err)
	}

	normalized, err := normalizeMarkdownSkillParameters(`Never include secrets in these parameters.
Required:
- query: Search query
Optional:
- timeout_ms: Timeout in milliseconds`)
	if err != nil {
		t.Fatalf("unexpected normalization error: %v", err)
	}
	if normalized["format"] != "markdown" {
		t.Fatalf("expected markdown format, got %v", normalized["format"])
	}
	if normalized["secret_policy"] != "forbidden" {
		t.Fatalf("expected forbidden secret policy, got %v", normalized["secret_policy"])
	}
}

func TestNormalizeSkillParameterList(t *testing.T) {
	items, err := normalizeSkillParameterList(nil)
	if err != nil {
		t.Fatalf("expected nil list to normalize, got %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected empty normalized list, got %d", len(items))
	}

	_, err = normalizeSkillParameterList("bad")
	if !errors.Is(err, ErrInvalidAgentSkills) {
		t.Fatalf("expected ErrInvalidAgentSkills for bad list shape, got %v", err)
	}

	_, err = normalizeSkillParameterList([]any{
		map[string]any{"name": "query", "description": "Search query"},
		map[string]any{"name": "query", "description": "Duplicate"},
	})
	if !errors.Is(err, ErrInvalidAgentSkills) {
		t.Fatalf("expected ErrInvalidAgentSkills for duplicate names, got %v", err)
	}

	_, err = normalizeSkillParameterList([]any{map[string]any{"name": "query", "description": "api key value"}})
	if !errors.Is(err, ErrInvalidSkillDescription) {
		t.Fatalf("expected ErrInvalidSkillDescription for secret-like description, got %v", err)
	}

	normalized, err := normalizeSkillParameterList([]any{
		map[string]any{"name": "timeout_ms", "description": "Timeout"},
		map[string]any{"name": "query", "description": "Search query"},
	})
	if err != nil {
		t.Fatalf("unexpected normalize list error: %v", err)
	}
	if normalized[0]["name"] != "query" {
		t.Fatalf("expected sorted list by name, got first=%v", normalized[0]["name"])
	}
}

func TestNormalizeJSONSkillParameters(t *testing.T) {
	_, err := normalizeJSONSkillParameters(map[string]any{
		"required": []any{map[string]any{"name": "query", "description": "Search query"}},
	})
	if !errors.Is(err, ErrInvalidSkillDescription) {
		t.Fatalf("expected ErrInvalidSkillDescription without forbidden policy, got %v", err)
	}

	_, err = normalizeJSONSkillParameters(map[string]any{
		"secret_policy": "forbidden",
		"required":      "bad",
	})
	if !errors.Is(err, ErrInvalidAgentSkills) {
		t.Fatalf("expected ErrInvalidAgentSkills for bad required shape, got %v", err)
	}

	normalized, err := normalizeJSONSkillParameters(map[string]any{
		"secret_policy": "forbidden",
		"required": []any{
			map[string]any{"name": "query", "description": "Search query"},
		},
		"optional": []any{
			map[string]any{"name": "timeout_ms", "description": "Timeout"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected normalize json error: %v", err)
	}
	if normalized["format"] != "json" {
		t.Fatalf("expected json format, got %v", normalized["format"])
	}
}

func TestNormalizeFormattedSkillParameters(t *testing.T) {
	markdownBody := `Do not include secrets in payload.
Required:
- query: Search query`

	if _, err := normalizeFormattedSkillParameters("markdown", map[string]any{"schema": markdownBody}); err != nil {
		t.Fatalf("expected markdown schema string to normalize, got %v", err)
	}
	if _, err := normalizeFormattedSkillParameters("markdown", map[string]any{"body": markdownBody}); err != nil {
		t.Fatalf("expected markdown body to normalize, got %v", err)
	}

	jsonSchema := map[string]any{
		"secret_policy": "forbidden",
		"required":      []any{map[string]any{"name": "query", "description": "Search query"}},
	}
	if _, err := normalizeFormattedSkillParameters("json", map[string]any{"schema": jsonSchema}); err != nil {
		t.Fatalf("expected json schema object to normalize, got %v", err)
	}
	if _, err := normalizeFormattedSkillParameters("json", jsonSchema); err != nil {
		t.Fatalf("expected direct json schema keys to normalize, got %v", err)
	}

	if _, err := normalizeFormattedSkillParameters("unknown", map[string]any{}); !errors.Is(err, ErrInvalidAgentSkills) {
		t.Fatalf("expected ErrInvalidAgentSkills for unsupported format, got %v", err)
	}
}

func TestNormalizeSkillParametersDispatcher(t *testing.T) {
	if out, err := normalizeSkillParameters(nil); err != nil || out != nil {
		t.Fatalf("expected nil input to normalize to nil,nil, got out=%v err=%v", out, err)
	}

	_, err := normalizeSkillParameters(123)
	if !errors.Is(err, ErrInvalidAgentSkills) {
		t.Fatalf("expected ErrInvalidAgentSkills for unsupported type, got %v", err)
	}

	_, err = normalizeSkillParameters(`Never include secrets.
Required:
- query: Search query`)
	if err != nil {
		t.Fatalf("expected markdown string to normalize, got %v", err)
	}

	_, err = normalizeSkillParameters(map[string]any{
		"format": "json",
		"schema": map[string]any{
			"secret_policy": "forbidden",
			"required":      []any{map[string]any{"name": "query", "description": "Search query"}},
		},
	})
	if err != nil {
		t.Fatalf("expected formatted json map to normalize, got %v", err)
	}
}
