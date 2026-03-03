package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	ErrUnauthorizedHuman = errors.New("unauthorized human")
)

type HumanIdentity struct {
	Provider      string
	Subject       string
	Email         string
	EmailVerified bool
}

type HumanAuthProvider interface {
	Authenticate(*http.Request) (HumanIdentity, error)
	Name() string
}

func NewHumanAuthProviderFromEnv() HumanAuthProvider {
	name := strings.TrimSpace(strings.ToLower(os.Getenv("HUMAN_AUTH_PROVIDER")))
	switch name {
	case "supabase":
		return NewSupabaseAuthProvider(os.Getenv("SUPABASE_JWT_SECRET"))
	default:
		return NewDevHumanAuthProvider()
	}
}

type DevHumanAuthProvider struct{}

func NewDevHumanAuthProvider() *DevHumanAuthProvider {
	return &DevHumanAuthProvider{}
}

func (p *DevHumanAuthProvider) Name() string {
	return "dev"
}

func (p *DevHumanAuthProvider) Authenticate(r *http.Request) (HumanIdentity, error) {
	id := strings.TrimSpace(r.Header.Get("X-Human-Id"))
	if id == "" {
		return HumanIdentity{}, ErrUnauthorizedHuman
	}
	email := strings.TrimSpace(strings.ToLower(r.Header.Get("X-Human-Email")))
	if email == "" {
		email = id + "@local.dev"
	}
	return HumanIdentity{
		Provider:      p.Name(),
		Subject:       id,
		Email:         email,
		EmailVerified: true,
	}, nil
}

type SupabaseAuthProvider struct {
	jwtSecret []byte
}

func NewSupabaseAuthProvider(secret string) *SupabaseAuthProvider {
	return &SupabaseAuthProvider{jwtSecret: []byte(secret)}
}

func (p *SupabaseAuthProvider) Name() string {
	return "supabase"
}

func (p *SupabaseAuthProvider) Authenticate(r *http.Request) (HumanIdentity, error) {
	if len(p.jwtSecret) == 0 {
		return HumanIdentity{}, ErrUnauthorizedHuman
	}
	token, err := ExtractBearerToken(r.Header.Get("Authorization"))
	if err != nil {
		return HumanIdentity{}, ErrUnauthorizedHuman
	}
	claims, err := parseAndVerifyHS256JWT(token, p.jwtSecret)
	if err != nil {
		return HumanIdentity{}, ErrUnauthorizedHuman
	}

	sub, _ := claims["sub"].(string)
	email, _ := claims["email"].(string)
	emailVerified, _ := claims["email_verified"].(bool)
	if strings.TrimSpace(sub) == "" {
		return HumanIdentity{}, ErrUnauthorizedHuman
	}

	if exp, ok := claims["exp"]; ok {
		expUnix, convErr := toInt64(exp)
		if convErr != nil || time.Unix(expUnix, 0).Before(time.Now()) {
			return HumanIdentity{}, ErrUnauthorizedHuman
		}
	}

	return HumanIdentity{
		Provider:      p.Name(),
		Subject:       sub,
		Email:         strings.ToLower(strings.TrimSpace(email)),
		EmailVerified: emailVerified,
	}, nil
}

func parseAndVerifyHS256JWT(token string, secret []byte) (map[string]any, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("malformed token")
	}

	headerPayload := parts[0] + "." + parts[1]
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("decode signature: %w", err)
	}

	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(headerPayload))
	expected := mac.Sum(nil)
	if !hmac.Equal(sig, expected) {
		return nil, errors.New("invalid signature")
	}

	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("decode header: %w", err)
	}
	var header map[string]any
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, fmt.Errorf("parse header: %w", err)
	}
	if alg, _ := header["alg"].(string); alg != "HS256" {
		return nil, errors.New("unsupported jwt alg")
	}

	claimBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode claims: %w", err)
	}
	var claims map[string]any
	if err := json.Unmarshal(claimBytes, &claims); err != nil {
		return nil, fmt.Errorf("parse claims: %w", err)
	}
	return claims, nil
}

func toInt64(v any) (int64, error) {
	switch t := v.(type) {
	case float64:
		return int64(t), nil
	case int64:
		return t, nil
	case int:
		return int64(t), nil
	case string:
		return strconv.ParseInt(t, 10, 64)
	default:
		return 0, fmt.Errorf("unsupported numeric type %T", v)
	}
}
