package store

import "testing"

func TestSanitizeErrorTextRedactsBackendBodies(t *testing.T) {
	input := "list objects status 403: <Error><Code>SignatureDoesNotMatch</Code></Error>"
	if got := SanitizeErrorText(input); got != "authorization failed" {
		t.Fatalf("expected authorization failure summary, got %q", got)
	}
}

func TestSanitizeErrorTextKeepsSafeShortMessages(t *testing.T) {
	if got := SanitizeErrorText("enqueue unavailable"); got != "enqueue unavailable" {
		t.Fatalf("expected safe short error to remain visible, got %q", got)
	}
}
