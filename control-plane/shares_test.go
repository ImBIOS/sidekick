// SPDX-License-Identifier: FSL-1.1-ALv2
// Copyright 2026 ImBIOS — licensed under FSL-1.1-ALv2, see LICENSE.md.
package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// mintShare is the test helper: POST .../shares, return id + secret.
func mintShare(t *testing.T, srv *Server, kind, id, key, ttl string) (string, string) {
	t.Helper()
	mux := http.NewServeMux()
	srv.Routes(mux)
	var path string
	if kind == "sim" {
		path = "/v1/sims/" + id + "/shares"
	} else {
		path = "/v1/emus/" + id + "/shares"
	}
	req := httptestReq(http.MethodPost, path, key, `{"ttl":"`+ttl+`"}`)
	rec := httptestRec(mux, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("mint %s %s: got %d: %s", kind, id, rec.Code, rec.Body.String())
	}
	var out struct {
		ID         string `json:"id"`
		ShareToken string `json:"share_token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out.ShareToken, "sk_share_") || out.ID == "" {
		t.Fatalf("bad mint body: %+v", out)
	}
	return out.ID, out.ShareToken
}

func publicGet(t *testing.T, mux *http.ServeMux, path string) (int, []byte, http.Header) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	res := rec.Result()
	return res.StatusCode, rec.Body.Bytes(), res.Header
}

func httptestReq(method, path, key, body string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	return req
}

func httptestRec(mux *http.ServeMux, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestShareSimLifecycle(t *testing.T) {
	srv := testHostServer()
	mux := http.NewServeMux()
	srv.Routes(mux)

	// org_a creates a sim.
	rec := authed(t, srv, http.MethodPost, "/v1/sims", "key-a", `{"device":"iPhone 16"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create sim: %d", rec.Code)
	}
	var sim Sim
	if err := json.Unmarshal(rec.Body.Bytes(), &sim); err != nil {
		t.Fatal(err)
	}

	// Mint: 24h default when ttl omitted.
	req := httptestReq(http.MethodPost, "/v1/sims/"+sim.ID+"/shares", "key-a", `{}`)
	if rec := httptestRec(mux, req); rec.Code != http.StatusCreated {
		t.Fatalf("mint default ttl: %d %s", rec.Code, rec.Body.String())
	}
	sid, secret := mintShare(t, srv, "sim", sim.ID, "key-a", "1h")

	// Host posts state: tree, shot, logs, video.
	shot := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x01}
	hostCalls := []struct{ method, path, body string }{
		{http.MethodPut, "/v1/host/sims/" + sim.ID + "/tree", `{"data_b64":"` + base64.StdEncoding.EncodeToString([]byte(`{"nodes":[]}`)) + `"}`},
		{http.MethodPut, "/v1/host/sims/" + sim.ID + "/screenshot", `{"data_b64":"` + base64.StdEncoding.EncodeToString(shot) + `"}`},
		{http.MethodPost, "/v1/host/sims/" + sim.ID + "/logs", `{"lines":["boot ok"]}`},
		{http.MethodPost, "/v1/host/sims/" + sim.ID + "/recording", `{"data_b64":"` + base64.StdEncoding.EncodeToString([]byte("fake-mp4")) + `"}`},
	}
	for _, hc := range hostCalls {
		req := httptestReq(hc.method, hc.path, testHostToken, hc.body)
		if rec := httptestRec(mux, req); rec.Code != http.StatusOK {
			t.Fatalf("host %s %s: %d %s", hc.method, hc.path, rec.Code, rec.Body.String())
		}
	}

	// Public reads: page (HTML), JSON snapshot, sub-resources — no auth.
	if code, body, h := publicGet(t, mux, "/s/"+secret); code != http.StatusOK || !strings.Contains(h.Get("Content-Type"), "text/html") || !bytes.Contains(body, []byte(sim.ID)) {
		t.Errorf("guest page: %d ct=%q has-id=%v", code, h.Get("Content-Type"), bytes.Contains(body, []byte(sim.ID)))
	}
	if code, body, _ := publicGet(t, mux, "/s/"+secret+"?format=json"); code != http.StatusOK || !bytes.Contains(body, []byte(`"has_video":true`)) {
		t.Errorf("guest json: %d %s", code, body)
	}
	if code, body, h := publicGet(t, mux, "/s/"+secret+"/screenshot"); code != http.StatusOK || h.Get("Content-Type") != "image/png" || !bytes.Equal(body, shot) {
		t.Errorf("guest shot: %d ct=%q len=%d", code, h.Get("Content-Type"), len(body))
	}
	if code, body, _ := publicGet(t, mux, "/s/"+secret+"/tree"); code != http.StatusOK || string(body) != `{"nodes":[]}` {
		t.Errorf("guest tree: %d %s", code, body)
	}
	if code, body, _ := publicGet(t, mux, "/s/"+secret+"/logs?tail=5"); code != http.StatusOK || !bytes.Contains(body, []byte("boot ok")) {
		t.Errorf("guest logs: %d %s", code, body)
	}
	if code, body, h := publicGet(t, mux, "/s/"+secret+"/video"); code != http.StatusOK || h.Get("Content-Type") != "video/mp4" || string(body) != "fake-mp4" {
		t.Errorf("guest video: %d ct=%q %q", code, h.Get("Content-Type"), body)
	}

	// Instance tokens (sandboxed agents) cannot mint shares.
	tok, secret2, ok := srv.store.CreateInstanceToken("org_a", "ci", []string{"sims"}, time.Hour)
	_ = tok
	if !ok {
		t.Fatal("instance token mint")
	}
	req = httptestReq(http.MethodPost, "/v1/sims/"+sim.ID+"/shares", secret2, `{"ttl":"1h"}`)
	if rec := httptestRec(mux, req); rec.Code != http.StatusForbidden {
		t.Errorf("instance mint: got %d, want 403", rec.Code)
	}

	// Revoke kills it: public reads 404.
	req = httptestReq(http.MethodDelete, "/v1/sims/"+sim.ID+"/shares/"+sid, "key-a", "")
	if rec := httptestRec(mux, req); rec.Code != http.StatusNoContent {
		t.Fatalf("revoke: %d %s", rec.Code, rec.Body.String())
	}
	if code, _, _ := publicGet(t, mux, "/s/"+secret); code != http.StatusNotFound {
		t.Errorf("revoked read: got %d, want 404", code)
	}
}

func TestShareIsolationAndExpiry(t *testing.T) {
	srv := testHostServer()
	mux := http.NewServeMux()
	srv.Routes(mux)

	rec := authed(t, srv, http.MethodPost, "/v1/sims", "key-a", `{"device":"iPhone 16"}`)
	var sim Sim
	if err := json.Unmarshal(rec.Body.Bytes(), &sim); err != nil {
		t.Fatal(err)
	}
	_, secret := mintShare(t, srv, "sim", sim.ID, "key-a", "1h")

	// Foreign org cannot mint (404, not 403 — no oracle) and cannot revoke.
	if rec := authed(t, srv, http.MethodPost, "/v1/sims/"+sim.ID+"/shares", "key-b", `{"ttl":"1h"}`); rec.Code != http.StatusNotFound {
		t.Errorf("foreign mint: got %d, want 404", rec.Code)
	}
	if rec := authed(t, srv, http.MethodDelete, "/v1/sims/"+sim.ID+"/shares/shr_nope", "key-b", ""); rec.Code != http.StatusNotFound {
		t.Errorf("foreign revoke: got %d, want 404", rec.Code)
	}
	// Foreign org with a stolen token still reads (the share IS the capability)
	// — but forging a token 404s.
	if code, _, _ := publicGet(t, mux, "/s/"+"sk_share_"+strings.Repeat("0", 64)); code != http.StatusNotFound {
		t.Errorf("forged token: got %d, want 404", code)
	}
	_ = secret

	// Expired share 404s.
	rec = authed(t, srv, http.MethodPost, "/v1/emus", "key-a", `{"profile":"P"}`)
	var emu struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &emu); err != nil {
		t.Fatal(err)
	}
	_, expSecret := mintShare(t, srv, "emu", emu.ID, "key-a", "1ms")
	time.Sleep(10 * time.Millisecond)
	if code, _, _ := publicGet(t, mux, "/s/"+expSecret); code != http.StatusNotFound {
		t.Errorf("expired share: got %d, want 404", code)
	}

	// Emu shares serve plane-hosted blobs but 404 video (no plane MP4 path).
	_, emuSecret := mintShare(t, srv, "emu", emu.ID, "key-a", "1h")
	if code, _, _ := publicGet(t, mux, "/s/"+emuSecret+"/video"); code != http.StatusNotFound {
		t.Errorf("emu video: got %d, want 404", code)
	}
	if code, body, _ := publicGet(t, mux, "/s/"+emuSecret+"?format=json"); code != http.StatusOK || !bytes.Contains(body, []byte(`"kind":"emu"`)) {
		t.Errorf("emu json: %d %s", code, body)
	}

	// Bad TTLs rejected at mint.
	for _, ttl := range []string{`"forever"`, `"0s"`, `"-1h"`, `"800h"`} {
		req := httptestReq(http.MethodPost, "/v1/sims/"+sim.ID+"/shares", "key-a", `{"ttl":`+ttl+`}`)
		if rec := httptestRec(mux, req); rec.Code != http.StatusBadRequest {
			t.Errorf("ttl %s: got %d, want 400", ttl, rec.Code)
		}
	}
}

// TestShareConcurrency hammers mint/lookup/revoke under -race: 32 racers ×
// 25 iterations must never return a foreign resource or panic.
func TestShareConcurrency(t *testing.T) {
	srv := testHostServer()
	rec := authed(t, srv, http.MethodPost, "/v1/sims", "key-a", `{"device":"iPhone 16"}`)
	var sim Sim
	_ = json.Unmarshal(rec.Body.Bytes(), &sim)
	done := make(chan bool, 64)
	for i := 0; i < 32; i++ {
		go func() {
			defer func() { done <- true }()
			for j := 0; j < 25; j++ {
				sh, secret, ok := srv.store.CreateGuestShare("org_a", "sim", sim.ID, time.Hour)
				if !ok {
					t.Error("mint")
					return
				}
				got, ok := srv.store.LookupGuestShare(secret)
				if !ok || got.ResourceID != sim.ID || got.ID != sh.ID {
					t.Error("lookup mismatch")
					return
				}
				if !srv.store.RevokeGuestShare("org_a", sh.ID) {
					t.Error("revoke")
					return
				}
				if _, ok := srv.store.LookupGuestShare(secret); ok {
					t.Error("revoked still resolves")
					return
				}
			}
		}()
	}
	for i := 0; i < 32; i++ {
		<-done
	}
}
