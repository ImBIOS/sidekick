// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 ImBIOS — licensed under Apache-2.0, see LICENSE-APACHE.md.
package mcp

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ImBIOS/sidekick/internal/api"
)

// stubPlane implements just enough of the control-plane for MCP dispatch
// tests: fixed org, one sim lifecycle, instant-done actions.
func stubPlane(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	ok := func(w http.ResponseWriter, v any) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(v)
	}
	mux.HandleFunc("/v1/whoami", func(w http.ResponseWriter, r *http.Request) {
		ok(w, map[string]string{"org_id": "org_test"})
	})
	mux.HandleFunc("/v1/sims", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
			ok(w, map[string]any{"id": "sim_1", "status": "running"})
			return
		}
		ok(w, map[string]any{"sims": []any{map[string]any{"id": "sim_1"}}})
	})
	mux.HandleFunc("/v1/sims/sim_1", func(w http.ResponseWriter, r *http.Request) {
		ok(w, map[string]any{"id": "sim_1", "status": "running"})
	})
	mux.HandleFunc("/v1/sims/sim_1/tree", func(w http.ResponseWriter, r *http.Request) {
		ok(w, map[string]any{"root": map[string]string{"ref": "@e1"}})
	})
	mux.HandleFunc("/v1/sims/sim_1/actions", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		ok(w, map[string]any{"ok": true, "queued": 1, "action_ids": []string{"act_1"}})
	})
	mux.HandleFunc("/v1/sims/sim_1/actions/act_1", func(w http.ResponseWriter, r *http.Request) {
		ok(w, map[string]any{"action_id": "act_1", "done": true, "result": "tapped @e1"})
	})
	png, _ := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==")
	mux.HandleFunc("/v1/sims/sim_1/screenshot", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(png)
	})
	return httptest.NewServer(mux)
}

// session runs a scripted JSON-RPC session, returning decoded responses.
func session(t *testing.T, srv *Server, lines ...string) []rpcResponse {
	t.Helper()
	var in bytes.Buffer
	for _, l := range lines {
		in.WriteString(l + "\n")
	}
	var out bytes.Buffer
	if err := srv.Run(&in, &out); err != nil {
		t.Fatal(err)
	}
	var resps []rpcResponse
	for _, l := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		var r rpcResponse
		if err := json.Unmarshal([]byte(l), &r); err != nil {
			t.Fatalf("bad response line %q: %v", l, err)
		}
		resps = append(resps, r)
	}
	return resps
}

func rpc(t *testing.T, id any, method string, params any) string {
	t.Helper()
	raw, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params})
	return string(raw)
}

func TestHandshakeAndList(t *testing.T) {
	srv := New(api.New("http://x", "k"))
	resps := session(t, srv,
		rpc(t, 1, "initialize", map[string]any{}),
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		rpc(t, 2, "tools/list", map[string]any{}),
	)
	if len(resps) != 2 {
		t.Fatalf("got %d responses, want 2 (notification gets none)", len(resps))
	}
	initRes, _ := json.Marshal(resps[0].Result)
	if !strings.Contains(string(initRes), "2024-11-05") {
		t.Errorf("handshake: %s", initRes)
	}
	listJSON, _ := json.Marshal(resps[1].Result)
	var list struct {
		Tools []toolDef `json:"tools"`
	}
	if err := json.Unmarshal(listJSON, &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Tools) < 15 {
		t.Fatalf("got %d tools, want >= 15", len(list.Tools))
	}
	names := map[string]bool{}
	for _, td := range list.Tools {
		names[td.Name] = true
		if td.InputSchema["type"] != "object" {
			t.Errorf("tool %s: bad inputSchema", td.Name)
		}
	}
	for _, want := range []string{"sidekick_org", "sidekick_xcode_build", "sidekick_ios_create", "sidekick_ios_act", "sidekick_ios_action", "sidekick_ios_screenshot", "sidekick_android_create"} {
		if !names[want] {
			t.Errorf("missing tool %s", want)
		}
	}
}

func TestAgentLoop(t *testing.T) {
	ts := stubPlane(t)
	defer ts.Close()
	srv := New(api.New(ts.URL, "k"))
	call := func(id int, name string, args map[string]any) rpcResponse {
		resps := session(t, srv, rpc(t, id, "tools/call", map[string]any{"name": name, "arguments": args}))
		if len(resps) != 1 {
			t.Fatalf("%s: got %d responses", name, len(resps))
		}
		return resps[0]
	}
	contentText := func(r rpcResponse) string {
		raw, _ := json.Marshal(r.Result)
		var res struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
				Data string `json:"data"`
			} `json:"content"`
		}
		if err := json.Unmarshal(raw, &res); err != nil || len(res.Content) == 0 {
			t.Fatalf("bad content: %s", raw)
		}
		return res.Content[0].Text
	}

	if got := contentText(call(1, "sidekick_org", nil)); !strings.Contains(got, "org_test") {
		t.Errorf("org: %s", got)
	}
	if got := contentText(call(2, "sidekick_ios_create", nil)); !strings.Contains(got, "sim_1") {
		t.Errorf("create: %s", got)
	}
	if got := contentText(call(3, "sidekick_ios_ax_tree", map[string]any{"id": "sim_1"})); !strings.Contains(got, "@e1") {
		t.Errorf("tree: %s", got)
	}
	actRes := call(4, "sidekick_ios_act", map[string]any{"id": "sim_1", "tap": "@e1"})
	if got := contentText(actRes); !strings.Contains(got, "act_1") {
		t.Errorf("act: %s", got)
	}
	if got := contentText(call(5, "sidekick_ios_action", map[string]any{"sim_id": "sim_1", "action_id": "act_1"})); !strings.Contains(got, "tapped @e1") {
		t.Errorf("action poll: %s", got)
	}

	// Screenshot arrives as MCP image content.
	shot := call(6, "sidekick_ios_screenshot", map[string]any{"id": "sim_1"})
	raw, _ := json.Marshal(shot.Result)
	var img struct {
		Content []struct {
			Type     string `json:"type"`
			Data     string `json:"data"`
			MimeType string `json:"mimeType"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &img); err != nil || img.Content[0].Type != "image" || img.Content[0].MimeType != "image/png" || img.Content[0].Data == "" {
		t.Fatalf("screenshot content: %s", raw)
	}

	// Unknown tool -> JSON-RPC error, not a crash.
	bad := call(7, "sidekick_nope", nil)
	if bad.Error == nil || bad.Error.Code != -32602 {
		t.Errorf("unknown tool: %+v", bad.Error)
	}
	// Missing required arg -> JSON-RPC error.
	missing := call(8, "sidekick_ios_act", map[string]any{"tap": "@e1"})
	if missing.Error == nil {
		t.Errorf("missing id should error")
	}
}

func TestMalformedInput(t *testing.T) {
	srv := New(api.New("http://x", "k"))
	resps := session(t, srv, `not json at all`, rpc(t, 1, "no/such-method", nil))
	if len(resps) != 2 || resps[0].Error == nil || resps[0].Error.Code != -32700 {
		t.Errorf("parse error handling: %+v", resps[0].Error)
	}
	if resps[1].Error == nil || resps[1].Error.Code != -32601 {
		t.Errorf("unknown method handling: %+v", resps[1].Error)
	}
}
