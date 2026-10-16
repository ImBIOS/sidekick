// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 ImBIOS — licensed under Apache-2.0, see LICENSE-APACHE.md.
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/ImBIOS/sidekick/internal/api"
	"github.com/ImBIOS/sidekick/testrun"
	"github.com/spf13/cobra"
)

func execCommand(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput()
}

// tokenCmd mints short-lived, scope-limited keys for sandboxed agents:
// `sidekick token create --name ci --scope sims,emus --ttl 2h`.
func tokenCmd() *cobra.Command {
	tok := &cobra.Command{Use: "token", Short: "scoped per-instance API tokens"}
	var name, scopes, ttl string
	create := &cobra.Command{
		Use:   "create",
		Short: "mint a token (secret shown once)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" || scopes == "" || ttl == "" {
				return fmt.Errorf("give --name, --scope (builds|sims|emus, comma-separated) and --ttl (e.g. 2h, max 720h)")
			}
			var out struct {
				ID        string   `json:"id"`
				Token     string   `json:"token"`
				Scopes    []string `json:"scopes"`
				ExpiresAt string   `json:"expires_at"`
			}
			body := map[string]string{"name": name, "ttl": ttl}
			// Server validates scope names; split client-side for the payload.
			payload := map[string]any{"name": name, "scopes": strings.Split(scopes, ","), "ttl": ttl}
			_ = body
			if err := call(cmd, http.MethodPost, "/v1/tokens", payload, &out); err != nil {
				return err
			}
			fmt.Printf("id:         %s\ntoken:      %s\nscopes:     %s\nexpires_at: %s\n(shown once — store it in the sandbox env)\n",
				out.ID, out.Token, strings.Join(out.Scopes, ","), out.ExpiresAt)
			return nil
		},
	}
	create.Flags().StringVar(&name, "name", "", "token name, e.g. ci-run-4821")
	create.Flags().StringVar(&scopes, "scope", "", "comma-separated: builds,sims,emus")
	create.Flags().StringVar(&ttl, "ttl", "", "lifetime, e.g. 2h (max 720h)")
	list := &cobra.Command{
		Use:   "list",
		Short: "list tokens (secrets never shown again)",
		RunE: func(cmd *cobra.Command, args []string) error {
			var out any
			if err := call(cmd, http.MethodGet, "/v1/tokens", nil, &out); err != nil {
				return err
			}
			printJSON(out)
			return nil
		},
	}
	revoke := &cobra.Command{
		Use:   "revoke <id>",
		Short: "revoke a token immediately",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := call(cmd, http.MethodDelete, "/v1/tokens/"+args[0], nil, nil); err != nil {
				return err
			}
			fmt.Println("revoked " + args[0])
			return nil
		},
	}
	tok.AddCommand(create, list, revoke)
	return tok
}

// testCmd runs recorded UI tests: replay verbs over adb, resolve moved
// refs, diff screenshots, exit nonzero on failure (CI-ready).
func testCmd() *cobra.Command {
	tst := &cobra.Command{Use: "test", Short: "record, replay and verify UI tests"}
	tst.AddCommand(testRecordCmd(), testReplayCmd(), testDiffCmd())
	return tst
}

// testRecordCmd builds a recording from CLI verbs:
// sidekick test record --out login.json tap @e1 type "hi" shot
func testRecordCmd() *cobra.Command {
	var out, sim, emu string
	c := &cobra.Command{
		Use:   "record --out test.json [tap @e1] [type hi] [swipe up] [press back] [wait 500] [shot]",
		Short: "build a recording from verbs",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rec := &testrun.Recording{Version: 1}
			i := 0
			for i < len(args) {
				switch args[i] {
				case "tap", "type", "swipe", "press", "wait":
					if i+1 >= len(args) {
						return fmt.Errorf("%s needs a value", args[i])
					}
					step := testrun.Step{Action: args[i]}
					switch args[i] {
					case "tap":
						step.Ref = args[i+1]
					case "type":
						step.Text = args[i+1]
					case "swipe":
						step.Dir = args[i+1]
					case "press":
						step.Key = args[i+1]
					case "wait":
						var ms int
						if _, err := fmt.Sscanf(args[i+1], "%d", &ms); err != nil {
							return fmt.Errorf("wait needs milliseconds: %w", err)
						}
						step.Ms = ms
					}
					rec.Steps = append(rec.Steps, step)
					i += 2
				case "shot":
					shot, err := fetchShot(cmd, sim, emu)
					if err != nil {
						return err
					}
					rec.Steps = append(rec.Steps, testrun.Step{Action: "tap", Ref: "", Shot: string(shot)})
					i++
				default:
					return fmt.Errorf("unknown verb %q (tap|type|swipe|press|wait|shot)", args[i])
				}
			}
			if out == "" {
				return fmt.Errorf("give --out test.json")
			}
			if err := rec.Save(out); err != nil {
				return err
			}
			fmt.Printf("recorded %d steps to %s\n", len(rec.Steps), out)
			return nil
		},
	}
	c.Flags().StringVar(&out, "out", "", "recording file")
	c.Flags().StringVar(&sim, "sim", "", "sim id for shot capture")
	c.Flags().StringVar(&emu, "emu", "", "emu id for shot capture")
	return c
}

func fetchShot(cmd *cobra.Command, sim, emu string) ([]byte, error) {
	switch {
	case sim != "":
		return callBytes(cmd, http.MethodGet, "/v1/sims/"+sim+"/screenshot", nil)
	case emu != "":
		return callBytes(cmd, http.MethodGet, "/v1/emus/"+emu+"/screenshot", nil)
	default:
		return nil, fmt.Errorf("shot needs --sim or --emu")
	}
}

// adbRunner drives input verbs over a local adb serial (usually the
// adb-proxy tunnel on :5555); tree/shot come from the control-plane.
type adbRunner struct {
	adb    string
	serial string
	conn   apiConn
	kind   string // "sims" | "emus"
	id     string
}

type apiConn interface {
	Tree() ([]byte, error)
	Shot() ([]byte, error)
}

func (r *adbRunner) adbArgs(args ...string) []string {
	return append([]string{"-s", r.serial}, args...)
}

func (r *adbRunner) Tap(ref string) error {
	tree, err := r.conn.Tree()
	if err != nil {
		return fmt.Errorf("tree: %w", err)
	}
	x, y, err := refCenter(tree, ref)
	if err != nil {
		return err
	}
	return adbDo(r.adb, r.adbArgs("shell", "input", "tap", itoa2(x), itoa2(y))...)
}
func (r *adbRunner) Type(text string) error {
	return adbDo(r.adb, r.adbArgs("shell", "input", "text", adbEscape(text))...)
}
func (r *adbRunner) Swipe(dir string) error {
	w, h := r.screenSize()
	x1, y1, x2, y2 := swipeCoords(dir, w, h)
	return adbDo(r.adb, r.adbArgs("shell", "input", "swipe",
		itoa2(x1), itoa2(y1), itoa2(x2), itoa2(y2))...)
}
func (r *adbRunner) Press(key string) error {
	return adbDo(r.adb, r.adbArgs("shell", "input", "keyevent", androidKey(key))...)
}
func (r *adbRunner) Screenshot() ([]byte, error) { return r.conn.Shot() }
func (r *adbRunner) Tree() ([]byte, error)       { return r.conn.Tree() }

// screenSize queries the device resolution once per call (cheap, cached by
// adb daemon). Falls back to 1080x2400 when unparseable.
func (r *adbRunner) screenSize() (int, int) {
	out, err := execCommand(r.adb, r.adbArgs("shell", "wm", "size")...)
	if err != nil {
		return 1080, 2400
	}
	var w, h int
	for _, f := range strings.Fields(string(out)) {
		if strings.Contains(f, "x") {
			if _, err := fmt.Sscanf(f, "%dx%d", &w, &h); err == nil && w > 0 && h > 0 {
				return w, h
			}
		}
	}
	return 1080, 2400
}

// refToXY taps by coordinates when the ref carries them (@x,y); plain AX
// refs (@e5) cannot drive adb directly, so replay requires a resolve step
// first — Replay handles that via Tree + fuzzy matching when the ref is a
// coordinate pair the recorder embedded.
func refToXY(ref string) []string {
	if strings.HasPrefix(ref, "@") {
		if xy, ok := strings.CutPrefix(ref, "@"); ok {
			if parts := strings.Split(xy, ","); len(parts) == 2 {
				return parts
			}
		}
	}
	return []string{ref}
}

// --- coordinate resolution + replay/diff commands ---

// refCenter resolves an AX ref to screen-center pixels by walking any tree
// shape (emu {nodes} or sim {root}) for {ref, bounds} pairs.
func refCenter(tree []byte, ref string) (int, int, error) {
	var doc any
	if err := json.Unmarshal(tree, &doc); err != nil {
		return 0, 0, fmt.Errorf("bad tree: %w", err)
	}
	bounds, ok := findBounds(doc, ref)
	if !ok {
		return 0, 0, fmt.Errorf("ref %s not on screen; re-read the tree", ref)
	}
	// "[x1,y1][x2,y2]"
	var x1, y1, x2, y2 int
	if _, err := fmt.Sscanf(bounds, "[%d,%d][%d,%d]", &x1, &y1, &x2, &y2); err != nil {
		return 0, 0, fmt.Errorf("bad bounds %q", bounds)
	}
	return (x1 + x2) / 2, (y1 + y2) / 2, nil
}

func findBounds(v any, ref string) (string, bool) {
	m, ok := v.(map[string]any)
	if !ok {
		return "", false
	}
	if r, _ := m["ref"].(string); r == ref {
		if b, _ := m["bounds"].(string); b != "" {
			return b, true
		}
	}
	for _, val := range m {
		switch t := val.(type) {
		case map[string]any:
			if b, ok := findBounds(t, ref); ok {
				return b, true
			}
		case []any:
			for _, e := range t {
				if b, ok := findBounds(e, ref); ok {
					return b, true
				}
			}
		}
	}
	return "", false
}

func adbDo(adb string, args ...string) error {
	out, err := execCommand(adb, args...)
	if err != nil {
		return fmt.Errorf("adb %s: %w (%s)", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

func adbEscape(text string) string {
	r := strings.NewReplacer(" ", "%s", "'", "\\'", "\"", "\\\"", "(", "\\(", ")", "\\)", "&", "\\&", ";", "\\;")
	return r.Replace(text)
}

func swipeCoords(dir string, w, h int) (int, int, int, int) {
	switch strings.ToLower(dir) {
	case "left":
		return w * 4 / 5, h / 2, w / 5, h / 2
	case "right":
		return w / 5, h / 2, w * 4 / 5, h / 2
	case "down":
		return w / 2, h / 5, w / 2, h * 4 / 5
	default: // up
		return w / 2, h * 4 / 5, w / 2, h / 5
	}
}

func androidKey(key string) string {
	switch strings.ToLower(key) {
	case "back":
		return "KEYCODE_BACK"
	case "home":
		return "KEYCODE_HOME"
	case "enter":
		return "KEYCODE_ENTER"
	}
	if strings.HasPrefix(strings.ToUpper(key), "KEYCODE_") {
		return strings.ToUpper(key)
	}
	return key
}

func itoa2(n int) string { return strconv.Itoa(n) }

// testReplayCmd replays a recording over adb and exits nonzero on failure.
func testReplayCmd() *cobra.Command {
	var in, sim, emu, serial, adbBin string
	var threshold float64
	var fuzzy bool
	var timeout time.Duration
	c := &cobra.Command{
		Use:   "replay --in test.json",
		Short: "replay a recording against a device (exit 1 on failure)",
		RunE: func(cmd *cobra.Command, args []string) error {
			rec, err := testrun.Load(in)
			if err != nil {
				return err
			}
			url, token := resolveConn(cmd)
			fetch := &apiFetcher{url: url, token: token, sim: sim, emu: emu}
			r := &adbRunner{adb: adbBin, serial: serial, conn: fetch}
			done := make(chan *testrun.Report, 1)
			go func() {
				done <- testrun.Replay(rec, r, testrun.ReplayOptions{DiffThreshold: threshold, Fuzzy: fuzzy})
			}()
			select {
			case rep := <-done:
				raw, _ := json.MarshalIndent(rep, "", "  ")
				fmt.Println(string(raw))
				if rep.Failed {
					return fmt.Errorf("replay failed")
				}
				fmt.Printf("passed %d steps\n", len(rep.Steps))
				return nil
			case <-time.After(timeout):
				return fmt.Errorf("replay timed out after %s", timeout)
			}
		},
	}
	c.Flags().StringVar(&in, "in", "", "recording file (required)")
	c.Flags().StringVar(&sim, "sim", "", "sim id for tree/screenshots")
	c.Flags().StringVar(&emu, "emu", "", "emu id for tree/screenshots")
	c.Flags().StringVar(&serial, "adb-serial", "localhost:5555", "adb target (adb-proxy tunnel)")
	c.Flags().StringVar(&adbBin, "adb", "adb", "adb binary")
	c.Flags().Float64Var(&threshold, "threshold", 0.05, "visual diff fail fraction")
	c.Flags().BoolVar(&fuzzy, "fuzzy", true, "re-resolve moved refs")
	c.Flags().DurationVar(&timeout, "timeout", 10*time.Minute, "max run time")
	return c
}

// apiFetcher serves tree/screenshots from the control-plane.
type apiFetcher struct {
	url, token, sim, emu string
}

func (f *apiFetcher) path(suffix string) (string, error) {
	switch {
	case f.sim != "":
		return "/v1/sims/" + f.sim + suffix, nil
	case f.emu != "":
		return "/v1/emus/" + f.emu + suffix, nil
	default:
		return "", fmt.Errorf("replay needs --sim or --emu for tree/screenshots")
	}
}

func (f *apiFetcher) Tree() ([]byte, error) {
	// Raw tree JSON (not decoded): resolution happens in refCenter.
	url, token := f.url, f.token
	path, err := f.path("/tree")
	if err != nil {
		return nil, err
	}
	raw, err := api.New(url, token).CallBytes(http.MethodGet, path, nil)
	return raw, err
}

func (f *apiFetcher) Shot() ([]byte, error) {
	url, token := f.url, f.token
	path, err := f.path("/screenshot")
	if err != nil {
		return nil, err
	}
	return api.New(url, token).CallBytes(http.MethodGet, path, nil)
}

// testDiffCmd compares two PNGs with a fail threshold.
func testDiffCmd() *cobra.Command {
	var threshold float64
	c := &cobra.Command{
		Use:   "diff <expected.png> <actual.png>",
		Short: "visual diff two screenshots (exit 1 over threshold)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}
			b, err := os.ReadFile(args[1])
			if err != nil {
				return err
			}
			pct, err := testrun.DiffPNG(a, b)
			if err != nil {
				return err
			}
			fmt.Printf("diff %.2f%% (threshold %.2f%%)\n", pct*100, threshold*100)
			if pct > threshold {
				os.Exit(1)
			}
			return nil
		},
	}
	c.Flags().Float64Var(&threshold, "threshold", 0.05, "fail fraction")
	return c
}
