// SPDX-License-Identifier: FSL-1.1-ALv2
// Copyright 2026 ImBIOS — licensed under FSL-1.1-ALv2, see LICENSE.md.
package main

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

func TestTokenLifecycle(t *testing.T) {
	srv := testHostServer()

	// Mint with org key.
	rec := authed(t, srv, http.MethodPost, "/v1/tokens", "key-a",
		`{"name":"ci-run","scopes":["sims","emus"],"ttl":"1h"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("mint: got %d: %s", rec.Code, rec.Body.String())
	}
	var minted struct {
		ID      string   `json:"id"`
		Token   string   `json:"token"`
		Scopes  []string `json:"scopes"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &minted); err != nil {
		t.Fatal(err)
	}
	if minted.Token == "" || len(minted.Scopes) != 2 {
		t.Fatalf("mint body: %+v", minted)
	}

	// Instance token reads sims (GET needs no scope).
	if rec := authed(t, srv, http.MethodGet, "/v1/sims", minted.Token, ""); rec.Code != http.StatusOK {
		t.Errorf("token read: got %d, want 200", rec.Code)
	}
	// ...writes within scope.
	if rec := authed(t, srv, http.MethodPost, "/v1/sims", minted.Token, `{"device":"iPhone 16"}`); rec.Code != http.StatusCreated {
		t.Errorf("token sim write: got %d, want 201", rec.Code)
	}
	// ...but builds are out of scope.
	if rec := authed(t, srv, http.MethodPost, "/v1/builds", minted.Token, `{"scheme":"X"}`); rec.Code != http.StatusForbidden {
		t.Errorf("out-of-scope write: got %d, want 403", rec.Code)
	}
	// ...and token management is org-key-only.
	if rec := authed(t, srv, http.MethodGet, "/v1/tokens", minted.Token, ""); rec.Code != http.StatusForbidden {
		t.Errorf("token managing tokens: got %d, want 403", rec.Code)
	}

	// Revoke kills it immediately.
	tokID := minted.ID
	if rec := authed(t, srv, http.MethodDelete, "/v1/tokens/"+tokID, "key-a", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("revoke: got %d", rec.Code)
	}
	if rec := authed(t, srv, http.MethodGet, "/v1/sims", minted.Token, ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("revoked token: got %d, want 401", rec.Code)
	}
}

func TestTokenValidation(t *testing.T) {
	srv := testHostServer()
	cases := []struct {
		name string
		body string
	}{
		{"no name", `{"scopes":["sims"],"ttl":"1h"}`},
		{"no scopes", `{"name":"x","ttl":"1h"}`},
		{"bad scope", `{"name":"x","scopes":["nukes"],"ttl":"1h"}`},
		{"dup scope", `{"name":"x","scopes":["sims","sims"],"ttl":"1h"}`},
		{"bad ttl", `{"name":"x","scopes":["sims"],"ttl":"forever"}`},
		{"huge ttl", `{"name":"x","scopes":["sims"],"ttl":"87600h"}`},
	}
	for _, tc := range cases {
		if rec := authed(t, srv, http.MethodPost, "/v1/tokens", "key-a", tc.body); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: got %d, want 400", tc.name, rec.Code)
		}
	}
}

func TestTokenExpiry(t *testing.T) {
	srv := testHostServer()
	rec := authed(t, srv, http.MethodPost, "/v1/tokens", "key-a",
		`{"name":"blink","scopes":["sims"],"ttl":"1ms"}`)
	var minted struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &minted); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
	if rec := authed(t, srv, http.MethodGet, "/v1/sims", minted.Token, ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("expired token: got %d, want 401", rec.Code)
	}
}

func TestPGTokenRoundtrip(t *testing.T) {
	s, mock := newMockStore(t)
	mock.ExpectExec("INSERT INTO instance_tokens").
		WithArgs(pgxmock.AnyArg(), "org_a", "ci", pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	tok, secret, ok := s.CreateInstanceToken("org_a", "ci", []string{"sims"}, time.Hour)
	if !ok || tok == nil || secret == "" {
		t.Fatalf("create: %+v %v", tok, ok)
	}
	future := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery("FROM instance_tokens WHERE hash=").
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"id", "org", "name", "scopes", "expires_at", "revoked", "created_at"}).
			AddRow(tok.ID, "org_a", "ci", []string{"sims"}, future, false, testTime))
	got, ok := s.LookupInstanceToken(secret)
	if !ok || got.ID != tok.ID || len(got.Scopes) != 1 {
		t.Errorf("lookup: %+v %v", got, ok)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}
