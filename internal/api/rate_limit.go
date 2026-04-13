package api

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	rateLimiterCleanupInterval = 256
	rateLimiterIdleTTL         = 10 * time.Minute
)

type rateLimiter struct {
	mu             sync.Mutex
	limitPerMinute int
	buckets        map[string]rateLimitBucket
	requestsSeen   uint64
}

type rateLimitBucket struct {
	tokens     float64
	lastRefill time.Time
	lastSeen   time.Time
}

type rateLimitDecision struct {
	allowed           bool
	retryAfterSeconds int
}

func newRateLimiter(limitPerMinute int) *rateLimiter {
	if limitPerMinute <= 0 {
		return nil
	}
	return &rateLimiter{
		limitPerMinute: limitPerMinute,
		buckets:        map[string]rateLimitBucket{},
	}
}

func withRateLimit(next http.Handler, limiter *rateLimiter, trustProxyHeaders bool) http.Handler {
	if limiter == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := clientIPForRateLimit(r, trustProxyHeaders)
		decision := limiter.allow(clientIP, time.Now().UTC())
		if decision.allowed {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Retry-After", strconv.Itoa(decision.retryAfterSeconds))
		writeErrorWithHintAndExtras(
			w,
			http.StatusTooManyRequests,
			"rate_limited",
			"rate limit exceeded for caller IP",
			nil,
			map[string]any{
				"failure":             true,
				"client_ip":           clientIP,
				"limit_per_minute":    limiter.limitPerMinute,
				"retry_after_seconds": decision.retryAfterSeconds,
			},
		)
	})
}

func (l *rateLimiter) allow(clientID string, now time.Time) rateLimitDecision {
	if l == nil || l.limitPerMinute <= 0 {
		return rateLimitDecision{allowed: true}
	}
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		clientID = "unknown"
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.requestsSeen++
	if l.requestsSeen%rateLimiterCleanupInterval == 0 {
		l.cleanupLocked(now)
	}

	bucket := l.buckets[clientID]
	if bucket.lastRefill.IsZero() {
		bucket.tokens = float64(l.limitPerMinute)
		bucket.lastRefill = now
	}

	refillPerSecond := float64(l.limitPerMinute) / 60.0
	if elapsed := now.Sub(bucket.lastRefill).Seconds(); elapsed > 0 {
		bucket.tokens = math.Min(float64(l.limitPerMinute), bucket.tokens+(elapsed*refillPerSecond))
		bucket.lastRefill = now
	}
	bucket.lastSeen = now

	if bucket.tokens >= 1 {
		bucket.tokens--
		l.buckets[clientID] = bucket
		return rateLimitDecision{allowed: true}
	}

	deficit := 1 - bucket.tokens
	retryAfterSeconds := int(math.Ceil(deficit / refillPerSecond))
	if retryAfterSeconds < 1 {
		retryAfterSeconds = 1
	}
	l.buckets[clientID] = bucket
	return rateLimitDecision{
		allowed:           false,
		retryAfterSeconds: retryAfterSeconds,
	}
}

func (l *rateLimiter) cleanupLocked(now time.Time) {
	if len(l.buckets) == 0 {
		return
	}
	cutoff := now.Add(-rateLimiterIdleTTL)
	for clientID, bucket := range l.buckets {
		if bucket.lastSeen.IsZero() || bucket.lastSeen.Before(cutoff) {
			delete(l.buckets, clientID)
		}
	}
}

func clientIPForRateLimit(r *http.Request, trustProxyHeaders bool) string {
	if r == nil {
		return "unknown"
	}
	if trustProxyHeaders {
		if forwarded := firstForwardedClientIP(r.Header.Get("X-Forwarded-For")); forwarded != "" {
			return forwarded
		}
		if realIP := normalizeClientIP(r.Header.Get("X-Real-IP")); realIP != "" {
			return realIP
		}
	}
	if remoteIP := normalizeClientIP(r.RemoteAddr); remoteIP != "" {
		return remoteIP
	}
	return "unknown"
}

func firstForwardedClientIP(raw string) string {
	for _, part := range strings.Split(raw, ",") {
		if candidate := normalizeClientIP(part); candidate != "" {
			return candidate
		}
	}
	return ""
}

func normalizeClientIP(raw string) string {
	candidate := strings.TrimSpace(raw)
	if candidate == "" {
		return ""
	}
	if host, _, err := net.SplitHostPort(candidate); err == nil {
		candidate = host
	}
	candidate = strings.Trim(candidate, "[]")
	if ip := net.ParseIP(candidate); ip != nil {
		return ip.String()
	}
	return ""
}
