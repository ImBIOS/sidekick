// Package main implements the Sidekick control-plane: org-scoped scheduling
// for remote Xcode builds, iOS simulators, and Android emulators.
//
// Auth model: agents call with `Authorization: Bearer <sk_live_...>` org API
// keys issued by the web console. Keys resolve to an org id via the web
// server's /internal/verify-key endpoint (WebVerifier), or via static
// SIDEKICK_API_KEYS entries for self-host/dev (StaticVerifier). Every
// resource is isolated by org id.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// OrgID identifies the tenant an agent key belongs to. All builds, sims, and
// emus are namespaced by it.
type OrgID string

// Verifier resolves a bearer API key to an org.
type Verifier interface {
	Verify(key string) (OrgID, error)
}

// StaticVerifier maps literal keys to orgs. Format:
//
//	SIDEKICK_API_KEYS="org_abc:sk_live_dev1,org_def:sk_live_dev2"
//
// Used for self-hosting, local dev, and tests.
type StaticVerifier struct {
	keys map[string]OrgID
}

// ParseStaticKeys parses SIDEKICK_API_KEYS. Empty input yields no keys.
func ParseStaticKeys(s string) *StaticVerifier {
	v := &StaticVerifier{keys: map[string]OrgID{}}
	for _, pair := range strings.Split(s, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		org, key, ok := strings.Cut(pair, ":")
		if !ok || org == "" || key == "" {
			continue
		}
		v.keys[strings.TrimSpace(key)] = OrgID(strings.TrimSpace(org))
	}
	return v
}

// Verify implements Verifier.
func (v *StaticVerifier) Verify(key string) (OrgID, error) {
	org, ok := v.keys[key]
	if !ok {
		return "", errUnauthorized
	}
	return org, nil
}

// WebVerifier delegates to the web server's /internal/verify-key endpoint,
// which checks better-auth API keys (hash compare, expiry, enabled flag).
// The shared CONTROL_PLANE_SECRET authenticates control-plane -> web.
type WebVerifier struct {
	baseURL string
	secret  string
	client  *http.Client
}

// NewWebVerifier builds a verifier against e.g. https://api.sidekick.imbios.dev.
func NewWebVerifier(baseURL, secret string) *WebVerifier {
	return &WebVerifier{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		secret:  secret,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

type verifyResponse struct {
	Valid bool   `json:"valid"`
	OrgID OrgID  `json:"orgId"`
	Error string `json:"error,omitempty"`
}

// Verify implements Verifier.
func (v *WebVerifier) Verify(key string) (OrgID, error) {
	body, _ := json.Marshal(map[string]string{"key": key})
	req, err := http.NewRequest(http.MethodPost, v.baseURL+"/internal/verify-key", strings.NewReader(string(body)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-control-plane-secret", v.secret)
	resp, err := v.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", errUnauthorized
	}
	var out verifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if !out.Valid || out.OrgID == "" {
		return "", errUnauthorized
	}
	return out.OrgID, nil
}

// ChainVerifier tries each verifier in order; the first success wins.
// Static keys (self-host) take precedence over the hosted web verifier.
type ChainVerifier struct {
	chain []Verifier
}

// Verify implements Verifier.
func (v ChainVerifier) Verify(key string) (OrgID, error) {
	for _, inner := range v.chain {
		if org, err := inner.Verify(key); err == nil {
			return org, nil
		}
	}
	return "", errUnauthorized
}

// verifierFromEnv builds the chain from the environment:
//
//	SIDEKICK_API_KEYS        static org:key pairs (self-host/dev)
//	SIDEKICK_AUTH_URL        web server base URL (hosted key verification)
//	CONTROL_PLANE_SECRET     shared secret for /internal/verify-key
func verifierFromEnv() Verifier {
	var chain []Verifier
	if static := ParseStaticKeys(os.Getenv("SIDEKICK_API_KEYS")); len(static.keys) > 0 {
		chain = append(chain, static)
	}
	if base, secret := os.Getenv("SIDEKICK_AUTH_URL"), os.Getenv("CONTROL_PLANE_SECRET"); base != "" && secret != "" {
		chain = append(chain, NewWebVerifier(base, secret))
	}
	if len(chain) == 0 {
		log.Println("control-plane: no verifiers configured (set SIDEKICK_API_KEYS or SIDEKICK_AUTH_URL+CONTROL_PLANE_SECRET); all requests will be rejected")
	}
	return ChainVerifier{chain: chain}
}
