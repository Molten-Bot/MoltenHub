package store

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const (
	awsV4Algorithm = "AWS4-HMAC-SHA256"
	awsV4Service   = "s3"
)

type s3Signer struct {
	accessKeyID     string
	secretAccessKey string
	region          string
	service         string
}

func newS3Signer(accessKeyID, secretAccessKey, region string) *s3Signer {
	accessKeyID = strings.TrimSpace(accessKeyID)
	secretAccessKey = strings.TrimSpace(secretAccessKey)
	region = strings.TrimSpace(region)
	if accessKeyID == "" || secretAccessKey == "" || region == "" {
		return nil
	}
	return &s3Signer{
		accessKeyID:     accessKeyID,
		secretAccessKey: secretAccessKey,
		region:          region,
		service:         awsV4Service,
	}
}

func (s *s3Signer) Sign(req *http.Request, payload []byte, now time.Time) error {
	if s == nil {
		return nil
	}
	if req == nil || req.URL == nil {
		return fmt.Errorf("sign request: nil request")
	}

	timestamp := now.UTC().Format("20060102T150405Z")
	shortDate := now.UTC().Format("20060102")

	payloadHash := sha256Hex(payload)
	req.Header.Set("x-amz-content-sha256", payloadHash)
	req.Header.Set("x-amz-date", timestamp)

	host := strings.ToLower(strings.TrimSpace(req.URL.Host))
	if host == "" {
		return fmt.Errorf("sign request: empty host")
	}
	req.Header.Set("Host", host)

	canonicalHeaders := "host:" + host + "\n" +
		"x-amz-content-sha256:" + payloadHash + "\n" +
		"x-amz-date:" + timestamp + "\n"
	signedHeaders := "host;x-amz-content-sha256;x-amz-date"

	canonicalRequest := strings.Join([]string{
		req.Method,
		canonicalURI(req.URL),
		canonicalQuery(req.URL.Query()),
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	scope := shortDate + "/" + s.region + "/" + s.service + "/aws4_request"
	stringToSign := strings.Join([]string{
		awsV4Algorithm,
		timestamp,
		scope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")

	signature := hex.EncodeToString(hmacSHA256(awsV4SigningKey(s.secretAccessKey, shortDate, s.region, s.service), stringToSign))
	req.Header.Set("Authorization",
		awsV4Algorithm+
			" Credential="+s.accessKeyID+"/"+scope+
			", SignedHeaders="+signedHeaders+
			", Signature="+signature)

	return nil
}

func awsV4SigningKey(secret, shortDate, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secret), shortDate)
	kRegion := hmacSHA256(kDate, region)
	kService := hmacSHA256(kRegion, service)
	return hmacSHA256(kService, "aws4_request")
}

func hmacSHA256(key []byte, data string) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(data))
	return mac.Sum(nil)
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func canonicalURI(u *url.URL) string {
	if u == nil {
		return "/"
	}
	escaped := u.EscapedPath()
	if escaped == "" {
		return "/"
	}
	if !strings.HasPrefix(escaped, "/") {
		return "/" + escaped
	}
	return escaped
}

func canonicalQuery(values url.Values) string {
	if len(values) == 0 {
		return ""
	}
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0)
	for _, k := range keys {
		vals := append([]string(nil), values[k]...)
		if len(vals) == 0 {
			parts = append(parts, awsPercentEncode(k)+"=")
			continue
		}
		sort.Strings(vals)
		encodedKey := awsPercentEncode(k)
		for _, v := range vals {
			parts = append(parts, encodedKey+"="+awsPercentEncode(v))
		}
	}
	return strings.Join(parts, "&")
}

func awsPercentEncode(s string) string {
	encoded := url.QueryEscape(s)
	encoded = strings.ReplaceAll(encoded, "+", "%20")
	encoded = strings.ReplaceAll(encoded, "*", "%2A")
	encoded = strings.ReplaceAll(encoded, "%7E", "~")
	return encoded
}
