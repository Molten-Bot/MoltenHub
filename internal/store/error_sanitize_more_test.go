package store

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestSanitizeErrorHandlesNilAndContextErrors(t *testing.T) {
	if got := SanitizeError(nil); got != "" {
		t.Fatalf("expected empty for nil error, got %q", got)
	}
	if got := SanitizeError(context.Canceled); got != "request canceled" {
		t.Fatalf("expected request canceled, got %q", got)
	}
	if got := SanitizeError(context.DeadlineExceeded); got != "request timed out" {
		t.Fatalf("expected request timed out, got %q", got)
	}

	wrappedCanceled := errors.Join(errors.New("outer"), context.Canceled)
	if got := SanitizeErrorWithDetail(wrappedCanceled); got != "request canceled" {
		t.Fatalf("expected joined canceled to sanitize, got %q", got)
	}
}

func TestSanitizeErrorTextBranchCoverage(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "connection", input: "dial tcp 127.0.0.1: connection refused", want: "connection failed"},
		{name: "not found", input: "request failed with status 404", want: "resource not found"},
		{name: "authorization", input: "status 403 signature mismatch", want: "authorization failed"},
		{name: "request failed fallback", input: "status 500 https://example.test/backend", want: "request failed"},
		{name: "safe short", input: "queue unavailable", want: "queue unavailable"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := SanitizeErrorText(tc.input); got != tc.want {
				t.Fatalf("SanitizeErrorText(%q)=%q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestSanitizeErrorDetailTextExtractsJSONFields(t *testing.T) {
	input := `status 503 body={"code":"AccessDenied","request_id":"req-123","cf-ray":"ray-456"}`
	got := SanitizeErrorDetailText(input)
	if got != "status=503, s3_code=AccessDenied, request_id=req-123, cf_ray=ray-456" {
		t.Fatalf("unexpected detail text: %q", got)
	}

	combined := SanitizeErrorTextWithDetail(input)
	if !strings.HasPrefix(combined, "request failed") {
		t.Fatalf("expected request failed summary, got %q", combined)
	}
	if !strings.Contains(combined, "request_id=req-123") {
		t.Fatalf("expected structured detail in combined output, got %q", combined)
	}
}

func TestSanitizeDiagnosticValueAndCode(t *testing.T) {
	if got := sanitizeDiagnosticValue("  req-123  "); got != "req-123" {
		t.Fatalf("expected normalized diagnostic value, got %q", got)
	}
	if got := sanitizeDiagnosticValue("invalid value!"); got != "" {
		t.Fatalf("expected invalid diagnostic value to be dropped, got %q", got)
	}
	if got := sanitizeDiagnosticValue(strings.Repeat("a", 121)); got != "" {
		t.Fatalf("expected overlong diagnostic value to be dropped, got %q", got)
	}

	if got := sanitizeDiagnosticCode("AccessDenied"); got != "AccessDenied" {
		t.Fatalf("expected valid diagnostic code, got %q", got)
	}
	if got := sanitizeDiagnosticCode("accessdenied"); got != "" {
		t.Fatalf("expected lowercase-only code to be dropped, got %q", got)
	}
	if got := sanitizeDiagnosticCode("Access_Denied"); got != "" {
		t.Fatalf("expected underscore code to be dropped, got %q", got)
	}
}
