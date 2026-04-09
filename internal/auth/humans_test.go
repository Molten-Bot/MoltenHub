package auth

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

type errReadCloser struct{}

func (errReadCloser) Read([]byte) (int, error) { return 0, errors.New("read failed") }
func (errReadCloser) Close() error             { return nil }

func TestNewHumanAuthProviderFromEnv(t *testing.T) {
	t.Setenv("HUMAN_AUTH_PROVIDER", "")
	if _, ok := NewHumanAuthProviderFromEnv().(*DevHumanAuthProvider); !ok {
		t.Fatal("expected default provider to be dev")
	}

	t.Setenv("HUMAN_AUTH_PROVIDER", "supabase")
	t.Setenv("SUPABASE_URL", " https://example.supabase.co/ ")
	t.Setenv("SUPABASE_ANON_KEY", " anon ")
	provider, ok := NewHumanAuthProviderFromEnv().(*SupabaseAuthProvider)
	if !ok {
		t.Fatal("expected supabase provider")
	}
	if provider.supabaseURL != "https://example.supabase.co" {
		t.Fatalf("unexpected normalized supabase URL: %q", provider.supabaseURL)
	}
	if provider.anonKey != "anon" {
		t.Fatalf("unexpected normalized anon key: %q", provider.anonKey)
	}
}

func TestDevHumanAuthProviderAuthenticate(t *testing.T) {
	provider := NewDevHumanAuthProvider()
	if provider.Name() != "dev" {
		t.Fatalf("expected dev provider name, got %q", provider.Name())
	}

	req, _ := http.NewRequest(http.MethodGet, "http://example.test", nil)
	if _, err := provider.Authenticate(req); !errors.Is(err, ErrUnauthorizedHuman) {
		t.Fatalf("expected unauthorized error, got %v", err)
	}

	req.Header.Set("X-Human-Id", " Alice ")
	identity, err := provider.Authenticate(req)
	if err != nil {
		t.Fatalf("unexpected auth error: %v", err)
	}
	if identity.Subject != "Alice" {
		t.Fatalf("unexpected subject: %q", identity.Subject)
	}
	if identity.Email != "Alice@local.dev" {
		t.Fatalf("unexpected fallback email: %q", identity.Email)
	}
	if !identity.EmailVerified {
		t.Fatal("expected fallback dev auth to mark email verified")
	}

	req.Header.Set("X-Human-Email", " USER@Example.COM ")
	identity, err = provider.Authenticate(req)
	if err != nil {
		t.Fatalf("unexpected auth error with explicit email: %v", err)
	}
	if identity.Email != "user@example.com" {
		t.Fatalf("expected normalized explicit email, got %q", identity.Email)
	}
}

func TestSupabaseAuthProviderAuthenticateErrors(t *testing.T) {
	request := func() *http.Request {
		req, _ := http.NewRequest(http.MethodGet, "http://example.test", nil)
		req.Header.Set("Authorization", "Bearer token-a")
		return req
	}

	provider := NewSupabaseAuthProvider("", "")
	if _, err := provider.Authenticate(request()); !errors.Is(err, ErrUnauthorizedHuman) {
		t.Fatalf("expected unauthorized for missing config, got %v", err)
	}

	provider = NewSupabaseAuthProvider("https://example.supabase.co", "anon")
	reqMissingBearer, _ := http.NewRequest(http.MethodGet, "http://example.test", nil)
	if _, err := provider.Authenticate(reqMissingBearer); !errors.Is(err, ErrUnauthorizedHuman) {
		t.Fatalf("expected unauthorized for missing bearer, got %v", err)
	}

	provider = NewSupabaseAuthProvider("::invalid::", "anon")
	if _, err := provider.Authenticate(request()); !errors.Is(err, ErrUnauthorizedHuman) {
		t.Fatalf("expected unauthorized for invalid supabase URL, got %v", err)
	}

	provider = NewSupabaseAuthProvider("https://example.supabase.co", "anon")
	provider.httpClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("dial failed")
	})}
	if _, err := provider.Authenticate(request()); !errors.Is(err, ErrUnauthorizedHuman) {
		t.Fatalf("expected unauthorized on transport error, got %v", err)
	}

	provider.httpClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusUnauthorized, Body: io.NopCloser(strings.NewReader(`{"error":"unauthorized"}`))}, nil
	})}
	if _, err := provider.Authenticate(request()); !errors.Is(err, ErrUnauthorizedHuman) {
		t.Fatalf("expected unauthorized for non-200 status, got %v", err)
	}

	provider.httpClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: errReadCloser{}}, nil
	})}
	if _, err := provider.Authenticate(request()); !errors.Is(err, ErrUnauthorizedHuman) {
		t.Fatalf("expected unauthorized for body read error, got %v", err)
	}

	provider.httpClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`not-json`))}, nil
	})}
	if _, err := provider.Authenticate(request()); !errors.Is(err, ErrUnauthorizedHuman) {
		t.Fatalf("expected unauthorized for invalid json, got %v", err)
	}

	provider.httpClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"email":"a@b.test"}`))}, nil
	})}
	if _, err := provider.Authenticate(request()); !errors.Is(err, ErrUnauthorizedHuman) {
		t.Fatalf("expected unauthorized when id missing, got %v", err)
	}
}

func TestSupabaseAuthProviderAuthenticateSuccess(t *testing.T) {
	provider := NewSupabaseAuthProvider("https://example.supabase.co", "anon")
	provider.httpClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != "https://example.supabase.co/auth/v1/user" {
			t.Fatalf("unexpected supabase URL: %s", req.URL.String())
		}
		if got := req.Header.Get("Authorization"); got != "Bearer token-a" {
			t.Fatalf("unexpected auth header: %q", got)
		}
		if got := req.Header.Get("apikey"); got != "anon" {
			t.Fatalf("unexpected apikey header: %q", got)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`{
				"id": " user-1 ",
				"email": " USER@Example.COM ",
				"email_confirmed_at": "2026-01-01T00:00:00Z"
			}`)),
		}, nil
	})}

	req, _ := http.NewRequest(http.MethodGet, "http://example.test", nil)
	req.Header.Set("Authorization", "Bearer token-a")
	identity, err := provider.Authenticate(req)
	if err != nil {
		t.Fatalf("unexpected auth error: %v", err)
	}
	if identity.Provider != "supabase" {
		t.Fatalf("expected provider supabase, got %q", identity.Provider)
	}
	if identity.Subject != "user-1" {
		t.Fatalf("expected normalized subject, got %q", identity.Subject)
	}
	if identity.Email != "user@example.com" {
		t.Fatalf("expected normalized email, got %q", identity.Email)
	}
	if !identity.EmailVerified {
		t.Fatal("expected email verified true")
	}

	provider.httpClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"id":"user-2","email":"user2@example.com","email_confirmed_at":""}`)),
		}, nil
	})}
	identity, err = provider.Authenticate(req)
	if err != nil {
		t.Fatalf("unexpected auth error on unverified email case: %v", err)
	}
	if identity.EmailVerified {
		t.Fatal("expected email verified false when email_confirmed_at is empty")
	}
}
