// SPDX-License-Identifier: FSL-1.1-ALv2
// Copyright 2026 ImBIOS — licensed under FSL-1.1-ALv2, see LICENSE.md.
package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// InstanceToken is a short-lived, scope-limited key for one sandboxed agent.
// A CI job mints `sk_inst_...` for exactly the resources its run needs;
// the key dies with the run (TTL) or by revocation. Unlike org keys, these
// can never manage tokens and can never cross scopes.
type InstanceToken struct {
	ID        string    `json:"id"`
	Org       OrgID     `json:"-"`
	Name      string    `json:"name"`
	Scopes    []string  `json:"scopes"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	Revoked   bool      `json:"-"`
}

const instancePrefix = "sk_inst_"

// ValidScopes are the resource families an instance token may touch.
var ValidScopes = []string{"builds", "sims", "emus"}

func validScope(s string) bool {
	for _, v := range ValidScopes {
		if s == v {
			return true
		}
	}
	return false
}

func mintSecret() (secret, hash string, err error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	secret = instancePrefix + hex.EncodeToString(raw)
	sum := sha256.Sum256([]byte(secret))
	return secret, hex.EncodeToString(sum[:]), nil
}

// scopeFor maps a write path to its required scope. Reads need no scope
// beyond a valid token; /v1/tokens/* always requires the org key itself.
func scopeFor(method, path string) (string, bool) {
	if method == http.MethodGet || method == http.MethodHead {
		return "", false
	}
	switch {
	case strings.HasPrefix(path, "/v1/builds"):
		return "builds", true
	case strings.HasPrefix(path, "/v1/sims"):
		return "sims", true
	case strings.HasPrefix(path, "/v1/emus"):
		return "emus", true
	default:
		return "", false
	}
}

// authz is the verified identity withAuthz attaches to the request.
type authz struct {
	org      OrgID
	scopes   map[string]bool // nil = org key (all scopes)
	instance bool
}

type authzKey struct{}

// withAuthz enforces org keys (full access) or instance tokens (scoped,
// unexpired, unrevoked). Token management endpoints reject instance tokens.
func (s *Server) withAuthz(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := bearerKey(r.Header.Get("Authorization"))
		if key == "" {
			writeErr(w, http.StatusUnauthorized, "missing bearer API key")
			return
		}
		if strings.HasPrefix(key, instancePrefix) {
			tok, ok := s.store.LookupInstanceToken(key)
			if !ok {
				writeErr(w, http.StatusUnauthorized, "invalid, expired, or revoked token")
				return
			}
			if strings.HasPrefix(r.URL.Path, "/v1/tokens") {
				writeErr(w, http.StatusForbidden, "instance tokens cannot manage tokens")
				return
			}
			if scope, needed := scopeFor(r.Method, r.URL.Path); needed && !tok.hasScope(scope) {
				writeErr(w, http.StatusForbidden, fmt.Sprintf("token lacks %q scope", scope))
				return
			}
			scopes := map[string]bool{}
			for _, sc := range tok.Scopes {
				scopes[sc] = true
			}
			next(w, r.WithContext(withAuthzCtx(r.Context(), authz{org: tok.Org, scopes: scopes, instance: true})))
			return
		}
		org, err := s.verifier.Verify(key)
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "invalid API key")
			return
		}
		next(w, r.WithContext(withAuthzCtx(withOrg(r.Context(), org), authz{org: org})))
	}
}

func (t *InstanceToken) hasScope(scope string) bool {
	for _, s := range t.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

func withAuthzCtx(ctx context.Context, a authz) context.Context {
	return context.WithValue(ctx, authzKey{}, a)
}

type createTokenReq struct {
	Name   string   `json:"name"`
	Scopes []string `json:"scopes"`
	TTL    string   `json:"ttl"`
}

// POST /v1/tokens — mint a scoped instance token (org key only; the secret
// is shown once). TTL like "24h", max 720h (30d).
func (s *Server) handleCreateToken(w http.ResponseWriter, r *http.Request) {
	var req createTokenReq
	if !decodeJSONBody(w, r, 64<<10, &req) {
		return
	}
	ttl, err := time.ParseDuration(req.TTL)
	if err != nil {
		writeErr(w, http.StatusBadRequest, `ttl must parse like "24h"`)
		return
	}
	tok, secret, ok := s.store.CreateInstanceToken(orgFromContext(r), req.Name, req.Scopes, ttl)
	if !ok {
		writeErr(w, http.StatusBadRequest, "need a name, 1+ valid scopes (builds|sims|emus), ttl 1m..720h")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"id": tok.ID, "token": secret, "name": tok.Name,
		"scopes": tok.Scopes, "expires_at": tok.ExpiresAt,
	})
}

// GET /v1/tokens — list org tokens (secrets never shown again).
func (s *Server) handleListTokens(w http.ResponseWriter, r *http.Request) {
	toks := s.store.ListInstanceTokens(orgFromContext(r))
	if toks == nil {
		toks = []*InstanceToken{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"tokens": toks})
}

// DELETE /v1/tokens/{id} — revoke immediately.
func (s *Server) handleRevokeToken(w http.ResponseWriter, r *http.Request) {
	if !s.store.RevokeInstanceToken(orgFromContext(r), r.PathValue("id")) {
		writeErr(w, http.StatusNotFound, "token not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
