package auth

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const googleCertsURL = "https://www.googleapis.com/oauth2/v1/certs"

type HTTPGoogleVerifier struct {
	client    *http.Client
	mu        sync.RWMutex
	keys      map[string]*rsa.PublicKey
	expiresAt time.Time
	now       func() time.Time
}

func NewHTTPGoogleVerifier(client *http.Client) *HTTPGoogleVerifier {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	return &HTTPGoogleVerifier{
		client: client,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

type googleJWTHeader struct {
	Alg string `json:"alg"`
	Kid string `json:"kid"`
	Typ string `json:"typ"`
}

type googleJWTPayload struct {
	Subject   string `json:"sub"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Picture   string `json:"picture"`
	Audience  string `json:"aud"`
	Issuer    string `json:"iss"`
	ExpiresAt int64  `json:"exp"`
}

func (v *HTTPGoogleVerifier) VerifyIDToken(ctx context.Context, idToken, audience string) (*GoogleIdentity, error) {
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid google token format")
	}

	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid token header: %w", err)
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid token payload: %w", err)
	}

	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid token signature: %w", err)
	}

	var header googleJWTHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, err
	}

	if header.Alg != "RS256" || header.Kid == "" {
		return nil, fmt.Errorf("unsupported google token header")
	}

	var payload googleJWTPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, err
	}

	if audience != "" && payload.Audience != audience {
		return nil, fmt.Errorf("unexpected google token audience")
	}
	if payload.Issuer != "accounts.google.com" && payload.Issuer != "https://accounts.google.com" {
		return nil, fmt.Errorf("unexpected google token issuer")
	}
	if payload.Subject == "" {
		return nil, fmt.Errorf("google token subject is missing")
	}
	if payload.ExpiresAt <= v.now().Unix() {
		return nil, fmt.Errorf("google token is expired")
	}

	publicKey, err := v.lookupKey(ctx, header.Kid)
	if err != nil {
		return nil, err
	}

	signed := []byte(parts[0] + "." + parts[1])
	hashed := sha256.Sum256(signed)
	if err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hashed[:], signature); err != nil {
		return nil, fmt.Errorf("invalid google token signature: %w", err)
	}

	return &GoogleIdentity{
		Subject:   payload.Subject,
		Email:     payload.Email,
		Name:      payload.Name,
		AvatarURL: payload.Picture,
	}, nil
}

func (v *HTTPGoogleVerifier) lookupKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.RLock()
	if key, ok := v.keys[kid]; ok && v.now().Before(v.expiresAt) {
		v.mu.RUnlock()
		return key, nil
	}
	v.mu.RUnlock()

	if err := v.refreshKeys(ctx); err != nil {
		return nil, err
	}

	v.mu.RLock()
	defer v.mu.RUnlock()
	key, ok := v.keys[kid]
	if !ok {
		return nil, fmt.Errorf("google signing key not found")
	}
	return key, nil
}

func (v *HTTPGoogleVerifier) refreshKeys(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, googleCertsURL, nil)
	if err != nil {
		return err
	}

	resp, err := v.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch google certs: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var certMap map[string]string
	if err := json.Unmarshal(body, &certMap); err != nil {
		return err
	}

	keys := make(map[string]*rsa.PublicKey, len(certMap))
	for kid, pemValue := range certMap {
		key, err := parseGooglePublicKey(pemValue)
		if err != nil {
			return err
		}
		keys[kid] = key
	}

	v.mu.Lock()
	defer v.mu.Unlock()
	v.keys = keys
	v.expiresAt = v.now().Add(parseGoogleCacheMaxAge(resp.Header.Get("Cache-Control")))
	return nil
}

func parseGooglePublicKey(certPEM string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return nil, errors.New("invalid google certificate pem")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}

	publicKey, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("google certificate is not RSA")
	}

	return publicKey, nil
}

func parseGoogleCacheMaxAge(cacheControl string) time.Duration {
	for _, part := range strings.Split(cacheControl, ",") {
		part = strings.TrimSpace(part)
		if !strings.HasPrefix(part, "max-age=") {
			continue
		}
		seconds, err := strconv.Atoi(strings.TrimPrefix(part, "max-age="))
		if err != nil {
			continue
		}
		return time.Duration(seconds) * time.Second
	}

	return time.Hour
}
