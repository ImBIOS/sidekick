// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 ImBIOS — licensed under Apache-2.0, see LICENSE-APACHE.md.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	flagURL    string
	flagToken  string
	httpClient = &http.Client{Timeout: 60 * time.Second}
)

// config is the local CLI profile written by `sidekick login`.
type config struct {
	Token   string `json:"token"`
	URL     string `json:"control_plane_url"`
	OrgID   string `json:"org_id"`
	Updated string `json:"updated_at"`
}

func configPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "sidekick", "config.json"), nil
}

func loadConfig() config {
	cfg := config{URL: "https://api.sidekick.imbios.dev"}
	if p, err := configPath(); err == nil {
		if raw, err := os.ReadFile(p); err == nil {
			_ = json.Unmarshal(raw, &cfg)
		}
	}
	if cfg.URL == "" {
		cfg.URL = "https://api.sidekick.imbios.dev"
	}
	return cfg
}

func saveConfig(cfg config) error {
	p, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	cfg.Updated = time.Now().UTC().Format(time.RFC3339)
	raw, _ := json.MarshalIndent(cfg, "", "  ")
	return os.WriteFile(p, raw, 0o600)
}

// resolveConn returns the effective control-plane URL and API key.
// Precedence: flags > env (SIDEKICK_URL/SIDEKICK_TOKEN) > config file.
func resolveConn(cmd *cobra.Command) (url, token string) {
	cfg := loadConfig()
	url = cfg.URL
	token = cfg.Token
	if v, _ := cmd.Flags().GetString("url"); v != "" {
		url = v
	} else if v := os.Getenv("SIDEKICK_URL"); v != "" {
		url = v
	}
	if v, _ := cmd.Flags().GetString("token"); v != "" {
		token = v
	} else if v := os.Getenv("SIDEKICK_TOKEN"); v != "" {
		token = v
	}
	return strings.TrimSuffix(url, "/"), token
}

type apiError struct {
	Status int
	Msg    string
}

func (e *apiError) Error() string { return fmt.Sprintf("sidekick: HTTP %d: %s", e.Status, e.Msg) }

// call performs an authenticated JSON request and decodes the JSON result
// into out (nil to discard). Non-2xx responses become *apiError.
func call(cmd *cobra.Command, method, path string, body any, out any) error {
	url, token := resolveConn(cmd)
	if token == "" {
		return fmt.Errorf("not logged in: run `sidekick login` or set SIDEKICK_TOKEN")
	}
	var rdr io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, url+path, rdr)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("control-plane unreachable at %s: %w", url, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(raw))
		var parsed map[string]string
		if json.Unmarshal(raw, &parsed) == nil && parsed["error"] != "" {
			msg = parsed["error"]
		}
		if msg == "" {
			msg = http.StatusText(resp.StatusCode)
		}
		return &apiError{Status: resp.StatusCode, Msg: msg}
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("bad response from control-plane: %w", err)
		}
	}
	return nil
}

func printJSON(v any) {
	raw, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(raw))
}

// callBytes performs an authenticated request and returns raw response bytes
// (for PNG screenshots and MP4 recordings). Non-2xx becomes *apiError.
func callBytes(cmd *cobra.Command, method, path string, body any) ([]byte, error) {
	url, token := resolveConn(cmd)
	if token == "" {
		return nil, fmt.Errorf("not logged in: run `sidekick login` or set SIDEKICK_TOKEN")
	}
	var rdr io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		rdr = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, url+path, rdr)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("control-plane unreachable at %s: %w", url, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(raw))
		var parsed map[string]string
		if json.Unmarshal(raw, &parsed) == nil && parsed["error"] != "" {
			msg = parsed["error"]
		}
		if msg == "" {
			msg = http.StatusText(resp.StatusCode)
		}
		return nil, &apiError{Status: resp.StatusCode, Msg: msg}
	}
	return raw, nil
}

var root = &cobra.Command{Use: "sidekick", Short: "Sidekick - remote Xcode, iOS, Android for cloud agents"}

func init() {
	root.PersistentFlags().StringVar(&flagURL, "url", "", "control-plane URL (or SIDEKICK_URL)")
	root.PersistentFlags().StringVar(&flagToken, "token", "", "org API key (or SIDEKICK_TOKEN)")
}

func main() {
	root.AddCommand(loginCmd(), logoutCmd(), orgCmd(), hostCmd(), xcodeCmd(), iosCmd(), androidCmd())
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

// --- hosts ---

type hostProfile struct {
	Control string `json:"control_plane_url"`
	Token   string `json:"host_token"`
	Pool    string `json:"pool"`
	Joined  string `json:"joined_at"`
}

func hostPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "sidekick", "host.json"), nil
}

// hostCmd joins a Mac/Linux machine to the fleet: the daemons read this
// profile (or --control/--token/--pool flags, or SIDEKICK_CONTROL/
// SIDEKICK_HOST_TOKEN/SIDEKICK_POOL env) to poll for claims.
func hostCmd() *cobra.Command {
	host := &cobra.Command{Use: "host", Short: "fleet host enrollment"}
	var control, token, pool string
	join := &cobra.Command{
		Use:   "join",
		Short: "save this machine's pool credentials for the daemons",
		RunE: func(cmd *cobra.Command, args []string) error {
			if control == "" {
				control = os.Getenv("SIDEKICK_CONTROL")
			}
			if token == "" {
				token = os.Getenv("SIDEKICK_HOST_TOKEN")
			}
			if control == "" || token == "" {
				return fmt.Errorf("give --control and --token (or SIDEKICK_CONTROL / SIDEKICK_HOST_TOKEN)")
			}
			if pool == "" {
				pool = "mac-pool"
			}
			p, err := hostPath()
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				return err
			}
			prof := hostProfile{
				Control: strings.TrimSuffix(control, "/"),
				Token:   token,
				Pool:    pool,
				Joined:  time.Now().UTC().Format(time.RFC3339),
			}
			raw, _ := json.MarshalIndent(prof, "", "  ")
			if err := os.WriteFile(p, raw, 0o600); err != nil {
				return err
			}
			fmt.Printf("joined pool %s via %s (daemons: build-daemon, sim-daemon)\n", pool, prof.Control)
			return nil
		},
	}
	join.Flags().StringVar(&control, "control", "", "control-plane URL")
	join.Flags().StringVar(&token, "token", "", "fleet host token (SIDEKICK_HOST_TOKEN)")
	join.Flags().StringVar(&pool, "pool", "", "pool name (default mac-pool)")
	host.AddCommand(join)
	return host
}

// --- auth ---

func loginCmd() *cobra.Command {
	var tokenFlag string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "save your org API key (from console Settings) and verify it",
		RunE: func(cmd *cobra.Command, args []string) error {
			token := tokenFlag
			if token == "" && len(args) > 0 {
				token = args[0]
			}
			if token == "" {
				fmt.Print("Paste your API key (Settings > API keys): ")
				line, err := bufio.NewReader(os.Stdin).ReadString('\n')
				if err != nil {
					return fmt.Errorf("reading key: %w", err)
				}
				token = strings.TrimSpace(line)
			}
			if token == "" {
				return fmt.Errorf("no API key given")
			}
			url, _ := resolveConn(cmd)
			cfg := loadConfig()
			cfg.Token = token
			if v, _ := cmd.Flags().GetString("url"); v != "" {
				cfg.URL = strings.TrimSuffix(v, "/")
				url = cfg.URL
			} else if v := os.Getenv("SIDEKICK_URL"); v != "" {
				url = strings.TrimSuffix(v, "/")
			}
			// Verify before saving: whoami resolves the key to its org.
			// Route the check through the --token flag so `call` picks it up.
			if err := cmd.Flags().Set("token", token); err != nil {
				return err
			}
			var who struct {
				OrgID string `json:"org_id"`
			}
			if err := call(cmd, http.MethodGet, "/v1/whoami", nil, &who); err != nil {
				return fmt.Errorf("key rejected: %w", err)
			}
			cfg.Token = token
			cfg.URL = url
			cfg.OrgID = who.OrgID
			if err := saveConfig(cfg); err != nil {
				return err
			}
			fmt.Printf("logged in to %s as org %s\n", url, who.OrgID)
			return nil
		},
	}
	cmd.Flags().StringVar(&tokenFlag, "token", "", "API key (avoids the prompt)")
	return cmd
}

func logoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "forget the saved API key",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := loadConfig()
			cfg.Token = ""
			cfg.OrgID = ""
			if err := saveConfig(cfg); err != nil {
				return err
			}
			fmt.Println("logged out")
			return nil
		},
	}
}

func orgCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "org",
		Short: "show the org your key belongs to",
		RunE: func(cmd *cobra.Command, args []string) error {
			var who struct {
				OrgID string `json:"org_id"`
			}
			if err := call(cmd, http.MethodGet, "/v1/whoami", nil, &who); err != nil {
				return err
			}
			fmt.Printf("org: %s\n", who.OrgID)
			return nil
		},
	}
}

// --- builds ---

func xcodeCmd() *cobra.Command {
	xcode := &cobra.Command{Use: "xcode", Short: "remote Xcode builds"}
	var scheme, ref string
	var wait bool
	build := &cobra.Command{
		Use:   "build [path]",
		Short: "queue a remote build and stream its logs",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if scheme == "" {
				return fmt.Errorf("--scheme is required")
			}
			path := first(args, ".")
			fmt.Printf("syncing %s -> control-plane, scheme=%s\n", path, scheme)
			var b struct {
				ID     string `json:"build_id"`
				Status string `json:"status"`
			}
			if err := call(cmd, http.MethodPost, "/v1/builds",
				map[string]string{"scheme": scheme, "ref": ref}, &b); err != nil {
				return err
			}
			fmt.Printf("build %s (%s)\n", b.ID, b.Status)
			if !wait {
				return nil
			}
			// Poll until terminal, printing new log lines as they arrive.
			seen := 0
			for {
				var cur struct {
					Status string `json:"status"`
				}
				if err := call(cmd, http.MethodGet, "/v1/builds/"+b.ID, nil, &cur); err != nil {
					return err
				}
				var logs struct {
					Lines []string `json:"lines"`
				}
				_ = call(cmd, http.MethodGet, "/v1/builds/"+b.ID+"/logs?format=json", nil, &logs)
				for ; seen < len(logs.Lines); seen++ {
					fmt.Println(logs.Lines[seen])
				}
				if cur.Status == "succeeded" || cur.Status == "failed" {
					fmt.Printf("build %s: %s\n", b.ID, cur.Status)
					if cur.Status == "failed" {
						os.Exit(1)
					}
					return nil
				}
				time.Sleep(2 * time.Second)
			}
		},
	}
	build.Flags().StringVar(&scheme, "scheme", "", "Xcode scheme (required)")
	build.Flags().StringVar(&ref, "ref", "", "git ref to build (default: working copy)")
	build.Flags().BoolVar(&wait, "wait", true, "stream logs until the build finishes")
	xcode.AddCommand(build)
	return xcode
}

// --- ios ---

func iosCmd() *cobra.Command {
	ios := &cobra.Command{Use: "ios", Short: "cloud iOS simulators"}

	var device, runtime string
	create := &cobra.Command{
		Use:   "create",
		Short: "create simulator, print id + webrtc/ax-tree URLs",
		RunE: func(cmd *cobra.Command, args []string) error {
			var sim struct {
				ID        string `json:"id"`
				Status    string `json:"status"`
				WebRTCURL string `json:"webrtc_url"`
				AXTreeURL string `json:"ax_tree_url"`
			}
			if err := call(cmd, http.MethodPost, "/v1/sims",
				map[string]string{"device": device, "runtime": runtime}, &sim); err != nil {
				return err
			}
			printJSON(sim)
			if sim.Status == "provisioning" {
				fmt.Println("provisioning: poll `sidekick ios get` until running, then `ax-tree` before acting")
			}
			return nil
		},
	}
	create.Flags().StringVar(&device, "device", "iPhone 16", "device name")
	create.Flags().StringVar(&runtime, "runtime", "", "iOS runtime (default: latest)")

	list := &cobra.Command{
		Use:   "list",
		Short: "list your org's simulators",
		RunE: func(cmd *cobra.Command, args []string) error {
			var out any
			if err := call(cmd, http.MethodGet, "/v1/sims", nil, &out); err != nil {
				return err
			}
			printJSON(out)
			return nil
		},
	}

	get := &cobra.Command{
		Use:   "get <id>",
		Short: "show one simulator (poll status until running)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var out any
			if err := call(cmd, http.MethodGet, "/v1/sims/"+args[0], nil, &out); err != nil {
				return err
			}
			printJSON(out)
			return nil
		},
	}

	del := &cobra.Command{
		Use:   "delete <id>",
		Short: "stop a simulator (billing stops)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := call(cmd, http.MethodDelete, "/v1/sims/"+args[0], nil, nil); err != nil {
				return err
			}
			fmt.Println("deleted " + args[0])
			return nil
		},
	}

	var tap, typ, swipe, batch string
	act := &cobra.Command{
		Use:   "act <id>",
		Short: "tap/type/swipe (AX refs from ax-tree)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]string{}
			if tap != "" {
				body["tap"] = tap
			}
			if typ != "" {
				body["type"] = typ
			}
			if swipe != "" {
				body["swipe"] = swipe
			}
			if batch != "" {
				raw, err := os.ReadFile(batch)
				if err != nil {
					return err
				}
				var actions []map[string]string
				if err := json.Unmarshal(raw, &actions); err != nil {
					return fmt.Errorf("bad batch file: %w", err)
				}
				var out any
				return call(cmd, http.MethodPost, "/v1/sims/"+args[0]+"/actions",
					map[string]any{"batch": actions}, &out)
			}
			if len(body) == 0 {
				return fmt.Errorf("give one of --tap, --type, --swipe, or --batch")
			}
			var out any
			if err := call(cmd, http.MethodPost, "/v1/sims/"+args[0]+"/actions", body, &out); err != nil {
				return err
			}
			printJSON(out)
			return nil
		},
	}
	act.Flags().StringVar(&tap, "tap", "", "AX ref to tap, e.g. @e5")
	act.Flags().StringVar(&typ, "type", "", "text to type")
	act.Flags().StringVar(&swipe, "swipe", "", "swipe direction")
	act.Flags().StringVar(&batch, "batch", "", "JSON file of actions for ms precision")

	ios.AddCommand(create, list, get, del, act, axTreeCmd(), shotCmd("/v1/sims/", "ios"), simLogsCmd(), recordCmd())
	return ios
}

// axTreeCmd prints the daemon-reported accessibility tree. Refs go stale
// after every screen change, so always re-read before acting.
func axTreeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ax-tree <id>",
		Short: "print the simulator's accessibility tree (@e refs)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var out any
			if err := call(cmd, http.MethodGet, "/v1/sims/"+args[0]+"/tree", nil, &out); err != nil {
				return err
			}
			printJSON(out)
			return nil
		},
	}
}

// shotCmd downloads the latest device screenshot. prefix selects sims/emus.
func shotCmd(prefix, what string) *cobra.Command {
	var out string
	c := &cobra.Command{
		Use:   "screenshot <id>",
		Short: "save the latest " + what + " screenshot",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			raw, err := callBytes(cmd, http.MethodGet, prefix+args[0]+"/screenshot", nil)
			if err != nil {
				return err
			}
			if out == "" {
				out = args[0] + ".png"
			}
			if err := os.WriteFile(out, raw, 0o644); err != nil {
				return err
			}
			fmt.Printf("saved %d bytes to %s\n", len(raw), out)
			return nil
		},
	}
	c.Flags().StringVar(&out, "out", "", "output file (default <id>.png)")
	return c
}

// simLogsCmd tails daemon device logs.
func simLogsCmd() *cobra.Command {
	var tail int
	c := &cobra.Command{
		Use:   "logs <id>",
		Short: "print simulator logs (crashes live here)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var out struct {
				Lines []string `json:"lines"`
			}
			path := "/v1/sims/" + args[0] + "/logs"
			if tail > 0 {
				path += "?tail=" + itoaInt(tail)
			}
			if err := call(cmd, http.MethodGet, path, nil, &out); err != nil {
				return err
			}
			for _, l := range out.Lines {
				fmt.Println(l)
			}
			return nil
		},
	}
	c.Flags().IntVar(&tail, "tail", 100, "last N lines")
	return c
}

// recordCmd starts/stops session recording. Stop downloads the MP4 for PRs.
func recordCmd() *cobra.Command {
	var stop bool
	var out string
	c := &cobra.Command{
		Use:   "record <id>",
		Short: "record the session; --stop ends it and saves the mp4",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			op := "start"
			if stop {
				op = "stop"
			}
			var res struct {
				Recording bool   `json:"recording"`
				VideoURL  string `json:"video_url"`
			}
			if err := call(cmd, http.MethodPost, "/v1/sims/"+args[0]+"/record", map[string]string{"op": op}, &res); err != nil {
				return err
			}
			if !stop {
				fmt.Println("recording started")
				return nil
			}
			raw, err := callBytes(cmd, http.MethodGet, res.VideoURL, nil)
			if err != nil {
				return fmt.Errorf("stopped, video not ready yet: %w", err)
			}
			if out == "" {
				out = args[0] + ".mp4"
			}
			if err := os.WriteFile(out, raw, 0o644); err != nil {
				return err
			}
			fmt.Printf("saved %d bytes to %s\n", len(raw), out)
			return nil
		},
	}
	c.Flags().BoolVar(&stop, "stop", false, "stop recording and download the mp4")
	c.Flags().StringVar(&out, "out", "", "output file (default <id>.mp4)")
	return c
}

// --- android ---

func androidCmd() *cobra.Command {
	android := &cobra.Command{Use: "android", Short: "cloud Android emulators"}

	var profile string
	create := &cobra.Command{
		Use:   "create",
		Short: "create emulator + print adb target",
		RunE: func(cmd *cobra.Command, args []string) error {
			var emu struct {
				ID        string `json:"id"`
				ADBTarget string `json:"adb_target"`
				Status    string `json:"status"`
			}
			if err := call(cmd, http.MethodPost, "/v1/emus",
				map[string]string{"profile": profile}, &emu); err != nil {
				return err
			}
			printJSON(emu)
			if emu.Status == "provisioning" {
				fmt.Println("provisioning: poll `sidekick android get` until running, then connect adb")
			}
			return nil
		},
	}
	create.Flags().StringVar(&profile, "profile", "Pixel_7_API_34", "emulator profile")

	list := &cobra.Command{
		Use:   "list",
		Short: "list your org's emulators",
		RunE: func(cmd *cobra.Command, args []string) error {
			var out any
			if err := call(cmd, http.MethodGet, "/v1/emus", nil, &out); err != nil {
				return err
			}
			printJSON(out)
			return nil
		},
	}

	get := &cobra.Command{
		Use:   "get <id>",
		Short: "show one emulator (poll status until running)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var out any
			if err := call(cmd, http.MethodGet, "/v1/emus/"+args[0], nil, &out); err != nil {
				return err
			}
			printJSON(out)
			return nil
		},
	}

	del := &cobra.Command{
		Use:   "delete <id>",
		Short: "stop an emulator (billing stops)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := call(cmd, http.MethodDelete, "/v1/emus/"+args[0], nil, nil); err != nil {
				return err
			}
			fmt.Println("deleted " + args[0])
			return nil
		},
	}

	tree := &cobra.Command{
		Use:   "ui-tree <id>",
		Short: "print the emulator's UI tree",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var out any
			if err := call(cmd, http.MethodGet, "/v1/emus/"+args[0]+"/tree", nil, &out); err != nil {
				return err
			}
			printJSON(out)
			return nil
		},
	}

	var tail int
	logs := &cobra.Command{
		Use:   "logs <id>",
		Short: "print emulator logcat lines",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var out struct {
				Lines []string `json:"lines"`
			}
			path := "/v1/emus/" + args[0] + "/logs"
			if tail > 0 {
				path += "?tail=" + itoaInt(tail)
			}
			if err := call(cmd, http.MethodGet, path, nil, &out); err != nil {
				return err
			}
			for _, l := range out.Lines {
				fmt.Println(l)
			}
			return nil
		},
	}
	logs.Flags().IntVar(&tail, "tail", 100, "last N lines")

	android.AddCommand(create, list, get, del, tree, shotCmd("/v1/emus/", "android"), logs)
	return android
}

func itoaInt(n int) string {
	return fmt.Sprintf("%d", n)
}

func first(s []string, d string) string {
	if len(s) > 0 {
		return s[0]
	}
	return d
}
