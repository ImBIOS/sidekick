// SPDX-License-Identifier: FSL-1.1-ALv2
// Copyright 2026 ImBIOS — licensed under FSL-1.1-ALv2, see LICENSE.md.
package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testServer() *Server {
	return NewServer(ParseStaticKeys("org_a:key-a,org_b:key-b"), NewStore())
}

func authed(t *testing.T, srv *Server, method, target, key, body string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	srv.Routes(mux)
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, target, reader)
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestAuthRequired(t *testing.T) {
	srv := testServer()
	for _, tc := range []struct {
		name   string
		header string
	}{
		{"missing", ""},
		{"garbage", "not-a-bearer"},
		{"unknown-key", "Bearer nope"},
	} {
		mux := http.NewServeMux()
		srv.Routes(mux)
		req := httptest.NewRequest(http.MethodGet, "/v1/whoami", nil)
		if tc.header != "" {
			req.Header.Set("Authorization", tc.header)
		}
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s: got %d, want 401", tc.name, rec.Code)
		}
	}
}

func TestWhoami(t *testing.T) {
	srv := testServer()
	rec := authed(t, srv, http.MethodGet, "/v1/whoami", "key-a", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", rec.Code)
	}
	var out map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["org_id"] != "org_a" {
		t.Errorf("org_id = %q, want org_a", out["org_id"])
	}
}

func TestOrgIsolation(t *testing.T) {
	srv := testServer()

	// org_a creates a sim.
	rec := authed(t, srv, http.MethodPost, "/v1/sims", "key-a", `{"device":"iPhone 16"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: got %d, want 201", rec.Code)
	}
	var sim Sim
	if err := json.Unmarshal(rec.Body.Bytes(), &sim); err != nil {
		t.Fatal(err)
	}

	// org_b cannot see it: get -> 404, list -> empty, delete -> 404.
	if rec := authed(t, srv, http.MethodGet, "/v1/sims/"+sim.ID, "key-b", ""); rec.Code != http.StatusNotFound {
		t.Errorf("cross-org get: got %d, want 404", rec.Code)
	}
	rec = authed(t, srv, http.MethodGet, "/v1/sims", "key-b", "")
	var list struct {
		Sims []Sim `json:"sims"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Sims) != 0 {
		t.Errorf("cross-org list: got %d sims, want 0", len(list.Sims))
	}
	if rec := authed(t, srv, http.MethodDelete, "/v1/sims/"+sim.ID, "key-b", ""); rec.Code != http.StatusNotFound {
		t.Errorf("cross-org delete: got %d, want 404", rec.Code)
	}

	// org_a still owns it, then deletes it.
	if rec := authed(t, srv, http.MethodGet, "/v1/sims/"+sim.ID, "key-a", ""); rec.Code != http.StatusOK {
		t.Errorf("own get: got %d, want 200", rec.Code)
	}
	if rec := authed(t, srv, http.MethodDelete, "/v1/sims/"+sim.ID, "key-a", ""); rec.Code != http.StatusNoContent {
		t.Errorf("own delete: got %d, want 204", rec.Code)
	}
}

func TestBuildValidation(t *testing.T) {
	srv := testServer()
	if rec := authed(t, srv, http.MethodPost, "/v1/builds", "key-a", `{}`); rec.Code != http.StatusBadRequest {
		t.Errorf("missing scheme: got %d, want 400", rec.Code)
	}
	rec := authed(t, srv, http.MethodPost, "/v1/builds", "key-a", `{"scheme":"MyApp"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: got %d, want 201", rec.Code)
	}
	var b Build
	if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil {
		t.Fatal(err)
	}
	if b.Status != StatusRunning {
		t.Errorf("status = %q, want running", b.Status)
	}
}

func TestParseStaticKeys(t *testing.T) {
	v := ParseStaticKeys("org_a:k1, org_b:k2 ,,badpair, :nokey")
	if org, err := v.Verify("k1"); err != nil || org != "org_a" {
		t.Errorf("k1 -> %q, %v", org, err)
	}
	if _, err := v.Verify("nope"); err == nil {
		t.Error("unknown key verified")
	}
	if len(ParseStaticKeys("").keys) != 0 {
		t.Error("empty input should yield no keys")
	}
}
