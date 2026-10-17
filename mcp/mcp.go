// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 ImBIOS — licensed under Apache-2.0, see LICENSE-APACHE.md.

// Package mcp exposes the Sidekick control-plane as a Model Context
// Protocol server over stdio (newline-delimited JSON-RPC). It speaks the
// stable MCP core: initialize, tools/list, tools/call. Every tool is a thin
// wrapper over internal/api, so CLI, skill, and MCP can never disagree on
// the wire contract. No third-party MCP dependency: the protocol surface
// we need fits in one file and stays dependency-free on purpose.
package mcp

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ImBIOS/sidekick/internal/api"
)

// Server is an MCP server bound to one control-plane connection.
type Server struct {
	API  api.Conn
	Name string
}

// New builds a server; empty name defaults to "sidekick".
func New(conn api.Conn) *Server {
	return &Server{API: conn, Name: "sidekick"}
}

// --- JSON-RPC plumbing ---

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type rpcResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id,omitempty"`
	Result  any    `json:"result,omitempty"`
	Error   *rpcError `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func errResp(id any, code int, format string, args ...any) rpcResponse {
	return rpcResponse{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: fmt.Sprintf(format, args...)}}
}

// Run serves one stdio session until EOF. Notifications get no reply.
func (s *Server) Run(in io.Reader, out io.Writer) error {
	w := bufio.NewWriter(out)
	defer w.Flush()
	sc := bufio.NewScanner(in)
	sc.Buffer(make([]byte, 8<<20), 8<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var req rpcRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			writeResp(w, errResp(nil, -32700, "parse error: %v", err))
			continue
		}
		if req.Method == "" || !strings.HasPrefix(req.Method, "notifications/") && req.ID == nil {
			writeResp(w, errResp(nil, -32600, "invalid request: missing id"))
			continue
		}
		if strings.HasPrefix(req.Method, "notifications/") {
			continue
		}
		writeResp(w, s.dispatch(req))
	}
	return sc.Err()
}

func writeResp(w *bufio.Writer, resp rpcResponse) {
	raw, _ := json.Marshal(resp)
	w.Write(raw)
	w.WriteByte('\n')
	w.Flush()
}

func (s *Server) dispatch(req rpcRequest) rpcResponse {
	switch req.Method {
	case "initialize":
		return rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": s.Name, "version": "1.0.0"},
		}}
	case "ping":
		return rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{}}
	case "tools/list":
		return rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{"tools": toolDefs()}}
	case "tools/call":
		var params struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return errResp(req.ID, -32602, "invalid params: %v", err)
		}
		if params.Arguments == nil {
			params.Arguments = map[string]any{}
		}
		result, isErr, err := s.callTool(params.Name, params.Arguments)
		if err != nil {
			return errResp(req.ID, -32602, "%v", err)
		}
		out := map[string]any{"content": result}
		if isErr {
			out["isError"] = true
		}
		return rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: out}
	default:
		return errResp(req.ID, -32601, "method not found: %s", req.Method)
	}
}

// --- tool definitions ---

type toolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

func strProp(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }

func objSchema(required []string, props map[string]any) map[string]any {
	return map[string]any{"type": "object", "properties": props, "required": required}
}

func toolDefs() []toolDef {
	req := func(names ...string) []string { return names }
	return []toolDef{
		{"sidekick_org", "Show the org the API key belongs to.", objSchema(nil, map[string]any{})},
		{"sidekick_xcode_build", "Queue a remote Xcode/Gradle build and optionally wait for it.", objSchema(req("scheme"), map[string]any{
			"scheme": strProp("Xcode scheme (required)"), "ref": strProp("git ref (default: working copy)"),
			"region": strProp("pin to a fleet region (default any)"),
			"wait": map[string]any{"type": "boolean", "description": "stream until terminal (default true)"},
		})},
		{"sidekick_ios_create", "Create a cloud iOS simulator. Poll sidekick_ios_get until running.", objSchema(nil, map[string]any{
			"device": strProp("device name (default iPhone 16)"), "runtime": strProp("iOS runtime (default latest)"),
			"region": strProp("pin to a fleet region (default any)"),
		})},
		{"sidekick_ios_get", "Show one simulator (status, URLs).", objSchema(req("id"), map[string]any{"id": strProp("sim id")})},
		{"sidekick_ios_list", "List your org's simulators.", objSchema(nil, map[string]any{})},
		{"sidekick_ios_delete", "Stop a simulator (billing stops).", objSchema(req("id"), map[string]any{"id": strProp("sim id")})},
		{"sidekick_ios_ax_tree", "Read the simulator accessibility tree. Always call before acting; refs go stale after each screen change.", objSchema(req("id"), map[string]any{"id": strProp("sim id")})},
		{"sidekick_ios_act", "Tap/type/swipe the simulator. Returns queued action IDs; poll sidekick_ios_action for results.", objSchema(req("id"), map[string]any{
			"id": strProp("sim id"), "tap": strProp("AX ref to tap, e.g. @e5"), "type": strProp("text to type"),
			"swipe": strProp("swipe direction"), "batch": map[string]any{"type": "array", "description": "actions for ms precision", "items": map[string]any{"type": "object"}},
		})},
		{"sidekick_ios_action", "Poll one queued verb until done (default 60s).", objSchema(req("sim_id", "action_id"), map[string]any{
			"sim_id": strProp("sim id"), "action_id": strProp("action id"),
			"timeout_s": map[string]any{"type": "number", "description": "max wait seconds (default 60)"},
		})},
		{"sidekick_ios_screenshot", "Capture the simulator screen (returned as an image).", objSchema(req("id"), map[string]any{"id": strProp("sim id")})},
		{"sidekick_ios_logs", "Read simulator logs (crashes live here).", objSchema(req("id"), map[string]any{
			"id": strProp("sim id"), "tail": map[string]any{"type": "number", "description": "last N lines (default 100)"},
		})},
		{"sidekick_ios_record", "Start/stop session recording. Stop notes the MP4 download path.", objSchema(req("id", "op"), map[string]any{
			"id": strProp("sim id"), "op": map[string]any{"type": "string", "description": `"start" or "stop"`, "enum": []string{"start", "stop"}},
		})},
		{"sidekick_android_create", "Create a cloud Android emulator. Poll sidekick_android_get until running.", objSchema(nil, map[string]any{
			"profile": strProp("emulator profile (default Pixel_7_API_34)"),
			"region": strProp("pin to a fleet region (default any)"),
		})},
		{"sidekick_android_get", "Show one emulator (status, adb target).", objSchema(req("id"), map[string]any{"id": strProp("emulator id")})},
		{"sidekick_android_list", "List your org's emulators.", objSchema(nil, map[string]any{})},
		{"sidekick_android_delete", "Stop an emulator (billing stops).", objSchema(req("id"), map[string]any{"id": strProp("emulator id")})},
		{"sidekick_android_ui_tree", "Read the emulator UI tree.", objSchema(req("id"), map[string]any{"id": strProp("emulator id")})},
		{"sidekick_android_screenshot", "Capture the emulator screen (returned as an image).", objSchema(req("id"), map[string]any{"id": strProp("emulator id")})},
		{"sidekick_android_logs", "Read emulator logcat lines.", objSchema(req("id"), map[string]any{
			"id": strProp("emulator id"), "tail": map[string]any{"type": "number", "description": "last N lines (default 100)"},
		})},
	}
}

// --- tool dispatch ---

func textContent(text string) []any {
	return []any{map[string]any{"type": "text", "text": text}}
}

func strArg(args map[string]any, key, dflt string) string {
	if v, ok := args[key].(string); ok && v != "" {
		return v
	}
	return dflt
}

func numArg(args map[string]any, key string, dflt float64) float64 {
	if v, ok := args[key].(float64); ok && v > 0 {
		return v
	}
	return dflt
}

func (s *Server) get(path string, out any) error {
	return s.API.Call(http.MethodGet, path, nil, out)
}

func (s *Server) callTool(name string, args map[string]any) ([]any, bool, error) {
	fail := func(format string, a ...any) ([]any, bool, error) {
		return textContent(fmt.Sprintf(format, a...)), true, nil
	}
	switch name {
	case "sidekick_org":
		var out any
		if err := s.get("/v1/whoami", &out); err != nil {
			return fail("%v", err)
		}
		return textContent(pretty(out)), false, nil

	case "sidekick_xcode_build":
		scheme := strArg(args, "scheme", "")
		if scheme == "" {
			return nil, false, fmt.Errorf("scheme is required")
		}
		body := map[string]string{"scheme": scheme, "ref": strArg(args, "ref", ""), "region": strArg(args, "region", "")}
		var b struct {
			ID     string `json:"build_id"`
			Status string `json:"status"`
		}
		if err := s.API.Call(http.MethodPost, "/v1/builds", body, &b); err != nil {
			return fail("%v", err)
		}
		wait := true
		if w, ok := args["wait"].(bool); ok {
			wait = w
		}
		if !wait {
			return textContent(pretty(b)), false, nil
		}
		deadline := time.Now().Add(10 * time.Minute)
		for {
			var cur struct {
				Status string `json:"status"`
			}
			if err := s.get("/v1/builds/"+b.ID, &cur); err != nil {
				return fail("%v", err)
			}
			if cur.Status == "succeeded" || cur.Status == "failed" {
				var logs struct {
					Lines []string `json:"lines"`
				}
				_ = s.get("/v1/builds/"+b.ID+"/logs?format=json", &logs)
				tail := logs.Lines
				if len(tail) > 50 {
					tail = tail[len(tail)-50:]
				}
				if cur.Status == "failed" {
					return textContent(fmt.Sprintf("build %s FAILED\n%s", b.ID, strings.Join(tail, "\n"))), true, nil
				}
				return textContent(fmt.Sprintf("build %s succeeded\n%s", b.ID, strings.Join(tail, "\n"))), false, nil
			}
			if time.Now().After(deadline) {
				return fail("build %s still %s after 10m; poll sidekick_ios_get-style status via CLI", b.ID, cur.Status)
			}
			time.Sleep(2 * time.Second)
		}

	case "sidekick_ios_create":
		var out any
		if err := s.API.Call(http.MethodPost, "/v1/sims", map[string]string{
			"device": strArg(args, "device", "iPhone 16"), "runtime": strArg(args, "runtime", ""), "region": strArg(args, "region", ""),
		}, &out); err != nil {
			return fail("%v", err)
		}
		return textContent(pretty(out) + "\nPoll sidekick_ios_get until status is running, then sidekick_ios_ax_tree before acting."), false, nil

	case "sidekick_ios_get":
		id, err := reqID(args, "id")
		if err != nil {
			return nil, false, err
		}
		var out any
		if err := s.get("/v1/sims/"+id, &out); err != nil {
			return fail("%v", err)
		}
		return textContent(pretty(out)), false, nil

	case "sidekick_ios_list":
		var out any
		if err := s.get("/v1/sims", &out); err != nil {
			return fail("%v", err)
		}
		return textContent(pretty(out)), false, nil

	case "sidekick_ios_delete":
		id, err := reqID(args, "id")
		if err != nil {
			return nil, false, err
		}
		if err := s.API.Call(http.MethodDelete, "/v1/sims/"+id, nil, nil); err != nil {
			return fail("%v", err)
		}
		return textContent("deleted " + id), false, nil

	case "sidekick_ios_ax_tree":
		id, err := reqID(args, "id")
		if err != nil {
			return nil, false, err
		}
		var out any
		if err := s.get("/v1/sims/"+id+"/tree", &out); err != nil {
			return fail("%v", err)
		}
		return textContent(pretty(out)), false, nil

	case "sidekick_ios_act":
		id, err := reqID(args, "id")
		if err != nil {
			return nil, false, err
		}
		body := map[string]any{}
		for _, k := range []string{"tap", "type", "swipe"} {
			if v := strArg(args, k, ""); v != "" {
				body[k] = v
			}
		}
		if b, ok := args["batch"]; ok {
			body["batch"] = b
		}
		if len(body) == 0 {
			return nil, false, fmt.Errorf("give one of tap, type, swipe, or batch")
		}
		var out any
		if err := s.API.Call(http.MethodPost, "/v1/sims/"+id+"/actions", body, &out); err != nil {
			return fail("%v", err)
		}
		return textContent(pretty(out) + "\nPoll sidekick_ios_action for each action_id."), false, nil

	case "sidekick_ios_action":
		simID, err := reqID(args, "sim_id")
		if err != nil {
			return nil, false, err
		}
		aid, err := reqID(args, "action_id")
		if err != nil {
			return nil, false, err
		}
		timeout := time.Duration(numArg(args, "timeout_s", 60)) * time.Second
		deadline := time.Now().Add(timeout)
		for {
			var a struct {
				Done   bool   `json:"done"`
				Result string `json:"result"`
			}
			if err := s.get("/v1/sims/"+simID+"/actions/"+aid, &a); err != nil {
				return fail("%v", err)
			}
			if a.Done {
				return textContent("done: " + a.Result), false, nil
			}
			if time.Now().After(deadline) {
				return fail("action %s not done after %ds; poll again", aid, int(timeout.Seconds()))
			}
			time.Sleep(time.Second)
		}

	case "sidekick_ios_screenshot":
		id, err := reqID(args, "id")
		if err != nil {
			return nil, false, err
		}
		raw, err := s.API.CallBytes(http.MethodGet, "/v1/sims/"+id+"/screenshot", nil)
		if err != nil {
			return fail("%v", err)
		}
		return []any{
			map[string]any{"type": "image", "data": base64.StdEncoding.EncodeToString(raw), "mimeType": "image/png"},
			map[string]any{"type": "text", "text": fmt.Sprintf("screenshot (%d bytes)", len(raw))},
		}, false, nil

	case "sidekick_ios_logs":
		id, err := reqID(args, "id")
		if err != nil {
			return nil, false, err
		}
		var out struct {
			Lines []string `json:"lines"`
		}
		if err := s.get(fmt.Sprintf("/v1/sims/%s/logs?tail=%d", id, int(numArg(args, "tail", 100))), &out); err != nil {
			return fail("%v", err)
		}
		return textContent(strings.Join(out.Lines, "\n")), false, nil

	case "sidekick_ios_record":
		id, err := reqID(args, "id")
		if err != nil {
			return nil, false, err
		}
		op := strArg(args, "op", "")
		if op != "start" && op != "stop" {
			return nil, false, fmt.Errorf(`op must be "start" or "stop"`)
		}
		var out any
		if err := s.API.Call(http.MethodPost, "/v1/sims/"+id+"/record", map[string]string{"op": op}, &out); err != nil {
			return fail("%v", err)
		}
		if op == "stop" {
			return textContent(pretty(out) + "\nDownload the MP4 with: sidekick ios record " + id + " --stop"), false, nil
		}
		return textContent(pretty(out)), false, nil

	case "sidekick_android_create":
		var out any
		if err := s.API.Call(http.MethodPost, "/v1/emus", map[string]string{
			"profile": strArg(args, "profile", "Pixel_7_API_34"), "region": strArg(args, "region", ""),
		}, &out); err != nil {
			return fail("%v", err)
		}
		return textContent(pretty(out) + "\nPoll sidekick_android_get until status is running, then connect adb."), false, nil

	case "sidekick_android_get":
		id, err := reqID(args, "id")
		if err != nil {
			return nil, false, err
		}
		var out any
		if err := s.get("/v1/emus/"+id, &out); err != nil {
			return fail("%v", err)
		}
		return textContent(pretty(out)), false, nil

	case "sidekick_android_list":
		var out any
		if err := s.get("/v1/emus", &out); err != nil {
			return fail("%v", err)
		}
		return textContent(pretty(out)), false, nil

	case "sidekick_android_delete":
		id, err := reqID(args, "id")
		if err != nil {
			return nil, false, err
		}
		if err := s.API.Call(http.MethodDelete, "/v1/emus/"+id, nil, nil); err != nil {
			return fail("%v", err)
		}
		return textContent("deleted " + id), false, nil

	case "sidekick_android_ui_tree":
		id, err := reqID(args, "id")
		if err != nil {
			return nil, false, err
		}
		var out any
		if err := s.get("/v1/emus/"+id+"/tree", &out); err != nil {
			return fail("%v", err)
		}
		return textContent(pretty(out)), false, nil

	case "sidekick_android_screenshot":
		id, err := reqID(args, "id")
		if err != nil {
			return nil, false, err
		}
		raw, err := s.API.CallBytes(http.MethodGet, "/v1/emus/"+id+"/screenshot", nil)
		if err != nil {
			return fail("%v", err)
		}
		return []any{
			map[string]any{"type": "image", "data": base64.StdEncoding.EncodeToString(raw), "mimeType": "image/png"},
			map[string]any{"type": "text", "text": fmt.Sprintf("screenshot (%d bytes)", len(raw))},
		}, false, nil

	case "sidekick_android_logs":
		id, err := reqID(args, "id")
		if err != nil {
			return nil, false, err
		}
		var out struct {
			Lines []string `json:"lines"`
		}
		if err := s.get(fmt.Sprintf("/v1/emus/%s/logs?tail=%d", id, int(numArg(args, "tail", 100))), &out); err != nil {
			return fail("%v", err)
		}
		return textContent(strings.Join(out.Lines, "\n")), false, nil

	default:
		return nil, false, fmt.Errorf("unknown tool: %s", name)
	}
}

func reqID(args map[string]any, key string) (string, error) {
	if v := strArg(args, key, ""); v != "" {
		return v, nil
	}
	return "", fmt.Errorf("%s is required", key)
}

func pretty(v any) string {
	raw, _ := json.MarshalIndent(v, "", "  ")
	return string(raw)
}
