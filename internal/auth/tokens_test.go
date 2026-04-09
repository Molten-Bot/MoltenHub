package auth

import (
	"encoding/base64"
	"testing"
)

func TestGenerateToken(t *testing.T) {
	token, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		t.Fatalf("token is not valid base64url: %v", err)
	}
	if len(decoded) != 32 {
		t.Fatalf("expected 32 random bytes, got %d", len(decoded))
	}
}

func TestHashToken(t *testing.T) {
	hashA := HashToken("abc")
	hashB := HashToken("abc")
	hashC := HashToken("def")

	if hashA == "" {
		t.Fatal("expected non-empty hash")
	}
	if hashA != hashB {
		t.Fatalf("expected deterministic hash, got %q vs %q", hashA, hashB)
	}
	if hashA == hashC {
		t.Fatalf("expected different inputs to produce different hashes")
	}
}

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name    string
		header  string
		want    string
		wantErr bool
	}{
		{name: "valid", header: "Bearer abc123", want: "abc123"},
		{name: "trim spaces", header: "Bearer   abc123   ", want: "abc123"},
		{name: "missing prefix", header: "Token abc", wantErr: true},
		{name: "empty token", header: "Bearer   ", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ExtractBearerToken(tc.header)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				if err != ErrMissingBearer {
					t.Fatalf("expected ErrMissingBearer, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("ExtractBearerToken(%q)=%q, want %q", tc.header, got, tc.want)
			}
		})
	}
}
