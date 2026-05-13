package model

import (
	"reflect"
	"testing"
)

func TestNormalizeAgentSkillName(t *testing.T) {
	if got, ok := NormalizeAgentSkillName(" Weather.Lookup "); !ok || got != "weather.lookup" {
		t.Fatalf("expected normalized skill name, got=%q ok=%v", got, ok)
	}
	if _, ok := NormalizeAgentSkillName("x"); ok {
		t.Fatal("expected one-character skill name to be invalid")
	}
	if _, ok := NormalizeAgentSkillName("bad skill"); ok {
		t.Fatal("expected skill name with spaces to be invalid")
	}
}

func TestNormalizeAgentSkillParameterName(t *testing.T) {
	if got, ok := NormalizeAgentSkillParameterName(" Param_Name-1 "); !ok || got != "param_name-1" {
		t.Fatalf("expected normalized parameter name, got=%q ok=%v", got, ok)
	}
	if _, ok := NormalizeAgentSkillParameterName("bad param"); ok {
		t.Fatal("expected parameter name with spaces to be invalid")
	}
}

func TestValidateSkillParameterPayloadKeys(t *testing.T) {
	got := ValidateSkillParameterPayloadKeys(map[string]string{
		"query":   "",
		"extra":   "x",
		"timeout": "10",
	}, []string{"query"}, map[string]struct{}{
		"query":   {},
		"timeout": {},
	})
	want := []string{
		`missing required parameter "query"`,
		`unknown parameter "extra"`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected validation errors: got=%v want=%v", got, want)
	}
}

func TestQuoteString(t *testing.T) {
	if got := QuoteString(`weather"lookup`); got != `"weather\"lookup"` {
		t.Fatalf("unexpected quoted string: %s", got)
	}
}
