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
)

const testHostToken = "host-secret"

func testHostServer() *Server {
	return NewServer(ParseStaticKeys("org_a:key-a,org_b:key-b"), NewStore()).WithHostToken(testHostToken)
}

func hosted(t *testing.T, srv *Server, method, target, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	srv.Routes(mux)
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestHostAuth(t *testing.T) {
	srv := testHostServer()
	// No token and wrong token both 401, even for valid paths.
	if rec := hosted(t, srv, http.MethodPost, "/v1/host/claims/next", "", `{}`); rec.Code != http.StatusUnauthorized {
		t.Errorf("no token: got %d, want 401", rec.Code)
	}
	if rec := hosted(t, srv, http.MethodPost, "/v1/host/claims/next", "wrong", `{}`); rec.Code != http.StatusUnauthorized {
		t.Errorf("wrong token: got %d, want 401", rec.Code)
	}
	// Org keys don't work on host endpoints either.
	if rec := hosted(t, srv, http.MethodPost, "/v1/host/claims/next", "key-a", `{"pool":"mac-pool"}`); rec.Code != http.StatusUnauthorized {
		t.Errorf("org key on host endpoint: got %d, want 401", rec.Code)
	}
	// Empty configured token rejects everything.
	bare := NewServer(ParseStaticKeys("org_a:key-a"), NewStore())
	if rec := hosted(t, bare, http.MethodPost, "/v1/host/claims/next", testHostToken, `{"pool":"x"}`); rec.Code != http.StatusUnauthorized {
		t.Errorf("unconfigured host token: got %d, want 401", rec.Code)
	}
}

func TestBuildClaimLifecycle(t *testing.T) {
	srv := testHostServer()
	// Agent queues a build.
	rec := authed(t, srv, http.MethodPost, "/v1/builds", "key-a", `{"scheme":"MyApp"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: got %d", rec.Code)
	}
	var b Build
	if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil {
		t.Fatal(err)
	}
	if b.Status != StatusQueued {
		t.Fatalf("status = %q, want queued", b.Status)
	}

	// Host asking for work it can't do gets 204.
	if rec := hosted(t, srv, http.MethodPost, "/v1/host/claims/next", testHostToken, `{"pool":"other-pool","kinds":["emu"]}`); rec.Code != http.StatusNoContent {
		t.Fatalf("empty claim: got %d, want 204", rec.Code)
	}

	// Host claims it: running + bound.
	rec = hosted(t, srv, http.MethodPost, "/v1/host/claims/next", testHostToken, `{"pool":"mac-1","kinds":["build"]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("claim: got %d", rec.Code)
	}
	var claim Claim
	if err := json.Unmarshal(rec.Body.Bytes(), &claim); err != nil {
		t.Fatal(err)
	}
	if claim.Kind != "build" || claim.Build.ID != b.ID || claim.Build.Status != StatusRunning || claim.Build.Host != "mac-1" {
		t.Fatalf("bad claim: %+v", claim)
	}

	// Daemon streams logs; agent reads them back.
	logs := `{"lines":["line one","line two"]}`
	if rec := hosted(t, srv, http.MethodPost, "/v1/host/builds/"+b.ID+"/logs", testHostToken, logs); rec.Code != http.StatusOK {
		t.Fatalf("append logs: got %d", rec.Code)
	}
	rec = authed(t, srv, http.MethodGet, "/v1/builds/"+b.ID+"/logs?format=json", "key-a", "")
	var doc struct {
		Lines  []string `json:"lines"`
		Status string   `json:"status"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Lines) != 4 || doc.Lines[2] != "line one" || doc.Status != StatusRunning {
		t.Errorf("unexpected log doc: %+v", doc)
	}

	// Daemon finishes; agent polls terminal state.
	if rec := hosted(t, srv, http.MethodPost, "/v1/host/builds/"+b.ID+"/finish", testHostToken, `{"status":"succeeded"}`); rec.Code != http.StatusOK {
		t.Fatalf("finish: got %d", rec.Code)
	}
	rec = authed(t, srv, http.MethodGet, "/v1/builds/"+b.ID, "key-a", "")
	var done Build
	if err := json.Unmarshal(rec.Body.Bytes(), &done); err != nil {
		t.Fatal(err)
	}
	if done.Status != StatusSucceeded {
		t.Errorf("status = %q, want succeeded", done.Status)
	}

	// Bad finish states rejected; unknown builds 404 on host paths.
	if rec := hosted(t, srv, http.MethodPost, "/v1/host/builds/"+b.ID+"/finish", testHostToken, `{"status":"exploded"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("bad status: got %d, want 400", rec.Code)
	}
	if rec := hosted(t, srv, http.MethodPost, "/v1/host/builds/nope/logs", testHostToken, logs); rec.Code != http.StatusNotFound {
		t.Errorf("unknown build logs: got %d, want 404", rec.Code)
	}
}

func TestSimProvisionAndVerbs(t *testing.T) {
	srv := testHostServer()
	// Agent creates: provisioning, no URLs yet.
	rec := authed(t, srv, http.MethodPost, "/v1/sims", "key-a", `{"device":"iPhone 16"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: got %d", rec.Code)
	}
	var sim Sim
	if err := json.Unmarshal(rec.Body.Bytes(), &sim); err != nil {
		t.Fatal(err)
	}
	if sim.Status != StatusProvisioning {
		t.Fatalf("status = %q, want provisioning", sim.Status)
	}

	// Tree/screenshot 404 before the daemon reports.
	if rec := authed(t, srv, http.MethodGet, "/v1/sims/"+sim.ID+"/tree", "key-a", ""); rec.Code != http.StatusNotFound {
		t.Errorf("early tree: got %d, want 404", rec.Code)
	}

	// Daemon claims + readies the sim.
	rec = hosted(t, srv, http.MethodPost, "/v1/host/claims/next", testHostToken, `{"pool":"mac-1","kinds":["sim"]}`)
	var claim Claim
	if err := json.Unmarshal(rec.Body.Bytes(), &claim); err != nil || claim.Kind != "sim" {
		t.Fatalf("sim claim: %d %v", rec.Code, err)
	}
	ready := `{"host":"mac-1","webrtc_url":"wss://x/` + sim.ID + `"}`
	if rec := hosted(t, srv, http.MethodPost, "/v1/host/sims/"+sim.ID+"/ready", testHostToken, ready); rec.Code != http.StatusOK {
		t.Fatalf("ready: got %d", rec.Code)
	}

	// Daemon posts tree; agent reads it back verbatim.
	tree := `{"root":{"ref":"@e1","label":"OK"}}`
	put := `{"data_b64":"` + base64.StdEncoding.EncodeToString([]byte(tree)) + `"}`
	if rec := hosted(t, srv, http.MethodPut, "/v1/host/sims/"+sim.ID+"/tree", testHostToken, put); rec.Code != http.StatusOK {
		t.Fatalf("put tree: got %d", rec.Code)
	}
	rec = authed(t, srv, http.MethodGet, "/v1/sims/"+sim.ID+"/tree", "key-a", "")
	if strings.TrimSpace(rec.Body.String()) != tree {
		t.Errorf("tree roundtrip: got %q", rec.Body.String())
	}
	// Invalid JSON rejected.
	bad := `{"data_b64":"` + base64.StdEncoding.EncodeToString([]byte("nope")) + `"}`
	if rec := hosted(t, srv, http.MethodPut, "/v1/host/sims/"+sim.ID+"/tree", testHostToken, bad); rec.Code != http.StatusBadRequest {
		t.Errorf("bad tree JSON: got %d, want 400", rec.Code)
	}

	// Screenshot roundtrip (1x1 PNG).
	png, _ := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==")
	sput := `{"data_b64":"` + base64.StdEncoding.EncodeToString(png) + `"}`
	if rec := hosted(t, srv, http.MethodPut, "/v1/host/sims/"+sim.ID+"/screenshot", testHostToken, sput); rec.Code != http.StatusOK {
		t.Fatalf("put shot: got %d", rec.Code)
	}
	rec = authed(t, srv, http.MethodGet, "/v1/sims/"+sim.ID+"/screenshot", "key-a", "")
	if rec.Code != http.StatusOK || !bytes.Equal(rec.Body.Bytes(), png) || rec.Header().Get("Content-Type") != "image/png" {
		t.Errorf("shot roundtrip: code=%d ct=%s len=%d", rec.Code, rec.Header().Get("Content-Type"), rec.Body.Len())
	}

	// Agent queues a tap; daemon drains it via claims; result poll works.
	rec = authed(t, srv, http.MethodPost, "/v1/sims/"+sim.ID+"/actions", "key-a", `{"tap":"@e1"}`)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("enqueue: got %d", rec.Code)
	}
	var enq struct {
		Queued    int      `json:"queued"`
		ActionIDs []string `json:"action_ids"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &enq); err != nil || enq.Queued != 1 {
		t.Fatalf("enqueue body: %+v %v", enq, err)
	}
	rec = hosted(t, srv, http.MethodPost, "/v1/host/claims/next", testHostToken, `{"pool":"mac-1","kinds":["sim-action"]}`)
	var aclaim Claim
	if err := json.Unmarshal(rec.Body.Bytes(), &aclaim); err != nil || aclaim.Kind != "sim-action" {
		t.Fatalf("action claim: %d %v", rec.Code, err)
	}
	if aclaim.Action.Body["tap"] != "@e1" {
		t.Errorf("action body: %+v", aclaim.Action.Body)
	}
	if rec := hosted(t, srv, http.MethodPost, "/v1/host/actions/"+aclaim.Action.ID+"/result", testHostToken, `{"result":"tapped @e1"}`); rec.Code != http.StatusOK {
		t.Fatalf("result: got %d", rec.Code)
	}
	rec = authed(t, srv, http.MethodGet, "/v1/sims/"+sim.ID+"/actions/"+aclaim.Action.ID, "key-a", "")
	var got QueuedAction
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || !got.Done || got.Result != "tapped @e1" {
		t.Errorf("action poll: %+v %v", got, err)
	}

	// Batch enqueues N; empty verbs rejected.
	rec = authed(t, srv, http.MethodPost, "/v1/sims/"+sim.ID+"/actions", "key-a", `{"batch":[{"tap":"@e1"},{"type":"hi"}]}`)
	if err := json.Unmarshal(rec.Body.Bytes(), &enq); err != nil || enq.Queued != 2 {
		t.Errorf("batch: %+v %v", enq, err)
	}
	if rec := authed(t, srv, http.MethodPost, "/v1/sims/"+sim.ID+"/actions", "key-a", `{}`); rec.Code != http.StatusBadRequest {
		t.Errorf("empty verb: got %d, want 400", rec.Code)
	}

	// Device logs + tail.
	if rec := hosted(t, srv, http.MethodPost, "/v1/host/sims/"+sim.ID+"/logs", testHostToken, `{"lines":["a","b","c"]}`); rec.Code != http.StatusOK {
		t.Fatalf("sim logs: got %d", rec.Code)
	}
	rec = authed(t, srv, http.MethodGet, "/v1/sims/"+sim.ID+"/logs?tail=2", "key-a", "")
	var logs struct {
		Lines []string `json:"lines"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &logs); err != nil || len(logs.Lines) != 2 || logs.Lines[0] != "b" {
		t.Errorf("tail: %+v %v", logs, err)
	}

	// Record start/stop + video download.
	if rec := authed(t, srv, http.MethodPost, "/v1/sims/"+sim.ID+"/record", "key-a", `{"op":"start"}`); rec.Code != http.StatusOK {
		t.Fatalf("record start: got %d", rec.Code)
	}
	vput := `{"data_b64":"` + base64.StdEncoding.EncodeToString([]byte("fake-mp4")) + `"}`
	if rec := hosted(t, srv, http.MethodPost, "/v1/host/sims/"+sim.ID+"/recording", testHostToken, vput); rec.Code != http.StatusOK {
		t.Fatalf("put video: got %d", rec.Code)
	}
	rec = authed(t, srv, http.MethodPost, "/v1/sims/"+sim.ID+"/record", "key-a", `{"op":"stop"}`)
	var stop struct {
		VideoURL string `json:"video_url"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &stop); err != nil || stop.VideoURL == "" {
		t.Fatalf("record stop: %+v %v", stop, err)
	}
	rec = authed(t, srv, http.MethodGet, "/v1/sims/"+sim.ID+"/recording/download", "key-a", "")
	if rec.Code != http.StatusOK || rec.Body.String() != "fake-mp4" {
		t.Errorf("video download: code=%d body=%q", rec.Code, rec.Body.String())
	}

	// Cross-org isolation holds on the new reads.
	if rec := authed(t, srv, http.MethodGet, "/v1/sims/"+sim.ID+"/tree", "key-b", ""); rec.Code != http.StatusNotFound {
		t.Errorf("cross-org tree: got %d, want 404", rec.Code)
	}
}

func TestEmuProvisionAndVerbs(t *testing.T) {
	srv := testHostServer()
	rec := authed(t, srv, http.MethodPost, "/v1/emus", "key-a", `{"profile":"Pixel_7"}`)
	var emu Emu
	if err := json.Unmarshal(rec.Body.Bytes(), &emu); err != nil {
		t.Fatal(err)
	}
	if emu.Status != StatusProvisioning || emu.ADBTarget != "" {
		t.Fatalf("emu fresh: %+v", emu)
	}
	rec = hosted(t, srv, http.MethodPost, "/v1/host/claims/next", testHostToken, `{"pool":"linux-1","kinds":["emu"]}`)
	var claim Claim
	if err := json.Unmarshal(rec.Body.Bytes(), &claim); err != nil || claim.Kind != "emu" {
		t.Fatalf("emu claim: %d %v", rec.Code, err)
	}
	if rec := hosted(t, srv, http.MethodPost, "/v1/host/emus/"+emu.ID+"/ready", testHostToken, `{"host":"linux-1","adb_target":"10.0.0.9:5555"}`); rec.Code != http.StatusOK {
		t.Fatalf("emu ready: got %d", rec.Code)
	}
	rec = authed(t, srv, http.MethodGet, "/v1/emus/"+emu.ID, "key-a", "")
	if err := json.Unmarshal(rec.Body.Bytes(), &emu); err != nil || emu.ADBTarget != "10.0.0.9:5555" || emu.Status != StatusRunning {
		t.Errorf("emu after ready: %+v %v", emu, err)
	}
	// Emu tree roundtrip.
	tree := `{"node":"root"}`
	put := `{"data_b64":"` + base64.StdEncoding.EncodeToString([]byte(tree)) + `"}`
	if rec := hosted(t, srv, http.MethodPut, "/v1/host/emus/"+emu.ID+"/tree", testHostToken, put); rec.Code != http.StatusOK {
		t.Fatalf("emu tree put: got %d", rec.Code)
	}
	rec = authed(t, srv, http.MethodGet, "/v1/emus/"+emu.ID+"/tree", "key-a", "")
	if strings.TrimSpace(rec.Body.String()) != tree {
		t.Errorf("emu tree: got %q", rec.Body.String())
	}
	// Emu logs.
	if rec := hosted(t, srv, http.MethodPost, "/v1/host/emus/"+emu.ID+"/logs", testHostToken, `{"lines":["boot complete"]}`); rec.Code != http.StatusOK {
		t.Fatalf("emu logs: got %d", rec.Code)
	}
}

func TestRegionPinning(t *testing.T) {
	srv := testHostServer()
	// One emu pinned to Tokyo, one unpinned.
	rec := authed(t, srv, http.MethodPost, "/v1/emus", "key-a", `{"profile":"P","region":"asia-northeast1"}`)
	var tokyo Emu
	if err := json.Unmarshal(rec.Body.Bytes(), &tokyo); err != nil {
		t.Fatal(err)
	}
	rec = authed(t, srv, http.MethodPost, "/v1/emus", "key-a", `{"profile":"P"}`)
	var anywhere Emu
	if err := json.Unmarshal(rec.Body.Bytes(), &anywhere); err != nil {
		t.Fatal(err)
	}

	// Singapore host sees only the unpinned emu.
	rec = hosted(t, srv, http.MethodPost, "/v1/host/claims/next", testHostToken, `{"pool":"linux-pool","region":"asia-southeast1","kinds":["emu"]}`)
	var c1 Claim
	if err := json.Unmarshal(rec.Body.Bytes(), &c1); err != nil || c1.Emu.ID != anywhere.ID {
		t.Fatalf("sg claim: %+v %v", c1, err)
	}
	if c1.Emu.Region != "asia-southeast1" {
		t.Errorf("unpinned emu should stick to claiming region, got %q", c1.Emu.Region)
	}
	// Tokyo host gets the pinned emu (and nothing else is left).
	rec = hosted(t, srv, http.MethodPost, "/v1/host/claims/next", testHostToken, `{"pool":"linux-pool","region":"asia-northeast1","kinds":["emu"]}`)
	var c2 Claim
	if err := json.Unmarshal(rec.Body.Bytes(), &c2); err != nil || c2.Emu.ID != tokyo.ID {
		t.Fatalf("tokyo claim: %+v %v", c2, err)
	}
	// Nothing left for anyone.
	if rec := hosted(t, srv, http.MethodPost, "/v1/host/claims/next", testHostToken, `{"pool":"linux-pool","region":"asia-northeast1","kinds":["emu"]}`); rec.Code != http.StatusNoContent {
		t.Errorf("drained pool: got %d, want 204", rec.Code)
	}
}
