// SPDX-License-Identifier: FSL-1.1-ALv2
// Copyright 2026 ImBIOS — licensed under FSL-1.1-ALv2, see LICENSE.md.

// Package fleet is the shared control-plane client for fleet daemons
// (build-daemon, sim-daemon, Linux emulator hosts). It speaks only the
// trusted /v1/host/* API with the SIDEKICK_HOST_TOKEN bearer token.
package fleet

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Client talks to one control-plane with one host token.
type Client struct {
	Control string
	Token   string
	HTTP    *http.Client
}

// Resolve fills control/token/pool from flags > host.json (sidekick host
// join) > env (SIDEKICK_CONTROL/SIDEKICK_HOST_TOKEN/SIDEKICK_POOL).
func Resolve(flagControl, flagToken, flagPool string) (control, token, pool string, err error) {
	control, token, pool = flagControl, flagToken, flagPool
	if (control == "" || token == "" || pool == "") && ifProfileExists() {
		if p, perr := loadProfile(); perr == nil {
			if control == "" {
				control = p.Control
			}
			if token == "" {
				token = p.Token
			}
			if pool == "" {
				pool = p.Pool
			}
		}
	}
	if control == "" {
		control = os.Getenv("SIDEKICK_CONTROL")
	}
	if token == "" {
		token = os.Getenv("SIDEKICK_HOST_TOKEN")
	}
	if pool == "" {
		pool = os.Getenv("SIDEKICK_POOL")
	}
	if control == "" || token == "" || pool == "" {
		return "", "", "", fmt.Errorf("need control+token+pool: flags, `sidekick host join`, or SIDEKICK_CONTROL/SIDEKICK_HOST_TOKEN/SIDEKICK_POOL")
	}
	return strings.TrimSuffix(control, "/"), token, pool, nil
}

type profile struct {
	Control string `json:"control_plane_url"`
	Token   string `json:"host_token"`
	Pool    string `json:"pool"`
}

func profilePath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "sidekick", "host.json"), nil
}

func ifProfileExists() bool {
	p, err := profilePath()
	if err != nil {
		return false
	}
	_, err = os.Stat(p)
	return err == nil
}

func loadProfile() (profile, error) {
	var p profile
	path, err := profilePath()
	if err != nil {
		return p, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return p, err
	}
	err = json.Unmarshal(raw, &p)
	return p, err
}

// New builds a client with a 60s HTTP timeout.
func New(control, token string) *Client {
	return &Client{Control: strings.TrimSuffix(control, "/"), Token: token, HTTP: &http.Client{Timeout: 60 * time.Second}}
}

// Claim is one unit of work. Kinds: build, sim, emu, sim-action.
type Claim struct {
	Kind   string         `json:"kind"`
	Build  map[string]any `json:"build,omitempty"`
	Sim    map[string]any `json:"sim,omitempty"`
	Emu    map[string]any `json:"emu,omitempty"`
	Action struct {
		ID    string            `json:"action_id"`
		SimID string            `json:"sim_id"`
		Body  map[string]string `json:"body"`
	} `json:"action,omitempty"`
}

func idOf(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

// BuildID/SimID/EmuID extract resource IDs from claim payloads.
func (c Claim) BuildID() string { return idOf(c.Build, "build_id", "id") }
func (c Claim) SimID() string   { return idOf(c.Sim, "id") }
func (c Claim) EmuID() string   { return idOf(c.Emu, "id") }

func (c *Client) do(method, path string, body any, out any) (int, error) {
	var rdr io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		rdr = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, c.Control+path, rdr)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, fmt.Errorf("sidekick: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return resp.StatusCode, err
		}
	}
	return resp.StatusCode, nil
}

// ClaimNext polls one claim for pool. ok=false means 204 (nothing to do).
func (c *Client) ClaimNext(pool string, kinds []string) (claim *Claim, ok bool, err error) {
	var cl Claim
	code, err := c.do(http.MethodPost, "/v1/host/claims/next", map[string]any{"pool": pool, "kinds": kinds}, &cl)
	if err != nil {
		if code == http.StatusNoContent {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &cl, true, nil
}

// PostBuildLogs streams daemon log lines.
func (c *Client) PostBuildLogs(id string, lines []string) error {
	_, err := c.do(http.MethodPost, "/v1/host/builds/"+id+"/logs", map[string]any{"lines": lines}, nil)
	return err
}

// FinishBuild marks a build succeeded or failed.
func (c *Client) FinishBuild(id, status string) error {
	_, err := c.do(http.MethodPost, "/v1/host/builds/"+id+"/finish", map[string]string{"status": status}, nil)
	return err
}

// SimReady marks a provisioned sim running.
func (c *Client) SimReady(id, host, webrtcURL string) error {
	_, err := c.do(http.MethodPost, "/v1/host/sims/"+id+"/ready",
		map[string]string{"host": host, "webrtc_url": webrtcURL}, nil)
	return err
}

// EmuReady marks a provisioned emu running.
func (c *Client) EmuReady(id, host, adbTarget string) error {
	_, err := c.do(http.MethodPost, "/v1/host/emus/"+id+"/ready",
		map[string]string{"host": host, "adb_target": adbTarget}, nil)
	return err
}

func b64(data []byte) map[string]string {
	return map[string]string{"data_b64": base64.StdEncoding.EncodeToString(data)}
}

// PutSimTree posts the AX tree JSON.
func (c *Client) PutSimTree(id string, tree []byte) error {
	_, err := c.do(http.MethodPut, "/v1/host/sims/"+id+"/tree", b64(tree), nil)
	return err
}

// PutSimShot posts the latest screenshot PNG.
func (c *Client) PutSimShot(id string, png []byte) error {
	_, err := c.do(http.MethodPut, "/v1/host/sims/"+id+"/screenshot", b64(png), nil)
	return err
}

// PostSimLogs streams device log lines.
func (c *Client) PostSimLogs(id string, lines []string) error {
	_, err := c.do(http.MethodPost, "/v1/host/sims/"+id+"/logs", map[string]any{"lines": lines}, nil)
	return err
}

// PostSimVideo posts the finished recording MP4.
func (c *Client) PostSimVideo(id string, mp4 []byte) error {
	_, err := c.do(http.MethodPost, "/v1/host/sims/"+id+"/recording", b64(mp4), nil)
	return err
}

// PostActionResult reports one applied verb.
func (c *Client) PostActionResult(actionID, result string) error {
	_, err := c.do(http.MethodPost, "/v1/host/actions/"+actionID+"/result", map[string]string{"result": result}, nil)
	return err
}

// PutEmuTree posts the UI tree JSON.
func (c *Client) PutEmuTree(id string, tree []byte) error {
	_, err := c.do(http.MethodPut, "/v1/host/emus/"+id+"/tree", b64(tree), nil)
	return err
}

// PutEmuShot posts the latest screenshot PNG.
func (c *Client) PutEmuShot(id string, png []byte) error {
	_, err := c.do(http.MethodPut, "/v1/host/emus/"+id+"/screenshot", b64(png), nil)
	return err
}

// PostEmuLogs streams logcat lines.
func (c *Client) PostEmuLogs(id string, lines []string) error {
	_, err := c.do(http.MethodPost, "/v1/host/emus/"+id+"/logs", map[string]any{"lines": lines}, nil)
	return err
}
