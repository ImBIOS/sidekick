// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 ImBIOS — licensed under Apache-2.0, see LICENSE-APACHE.md.
package testrun

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeRunner scripts device responses.
type fakeRunner struct {
	tree  string
	shot  []byte
	calls []string
	fail  map[string]error
}

func (f *fakeRunner) Tap(ref string) error {
	f.calls = append(f.calls, "tap "+ref)
	return f.fail["tap"]
}
func (f *fakeRunner) Type(text string) error {
	f.calls = append(f.calls, "type "+text)
	return f.fail["type"]
}
func (f *fakeRunner) Swipe(dir string) error {
	f.calls = append(f.calls, "swipe "+dir)
	return f.fail["swipe"]
}
func (f *fakeRunner) Press(key string) error {
	f.calls = append(f.calls, "press "+key)
	return f.fail["press"]
}
func (f *fakeRunner) Screenshot() ([]byte, error) { return f.shot, nil }
func (f *fakeRunner) Tree() ([]byte, error)       { return []byte(f.tree), nil }

func solidPNG(t *testing.T, w, h int, c color.Color) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestDiffIdentical(t *testing.T) {
	a := solidPNG(t, 8, 8, color.RGBA{10, 20, 30, 255})
	pct, err := DiffPNG(a, a)
	if err != nil || pct != 0 {
		t.Errorf("identical: pct=%v err=%v", pct, err)
	}
}

func TestDiffDetectsChange(t *testing.T) {
	a := solidPNG(t, 10, 10, color.RGBA{0, 0, 0, 255})
	// 1 white pixel in 100 -> 1%.
	mixed := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			mixed.Set(x, y, color.RGBA{0, 0, 0, 255})
		}
	}
	mixed.Set(0, 0, color.RGBA{255, 255, 255, 255})
	var buf2 bytes.Buffer
	if err := png.Encode(&buf2, mixed); err != nil {
		t.Fatal(err)
	}
	pct, err := DiffPNG(a, buf2.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if pct < 0.009 || pct > 0.011 {
		t.Errorf("pct = %v, want ~0.01", pct)
	}
}

func TestDiffSizeMismatch(t *testing.T) {
	a := solidPNG(t, 4, 4, color.RGBA{0, 0, 0, 255})
	b := solidPNG(t, 8, 8, color.RGBA{0, 0, 0, 255})
	pct, err := DiffPNG(a, b)
	if err != nil {
		t.Fatal(err)
	}
	// Overlap identical, overhang (64-16)/64 = 0.75 different.
	if pct < 0.74 || pct > 0.76 {
		t.Errorf("pct = %v, want ~0.75", pct)
	}
}

func TestResolveExactAndFuzzy(t *testing.T) {
	tree := `{"nodes":[{"ref":"@e1","class":"FrameLayout","children":[{"ref":"@e2","class":"Button","text":"Send"}]}]}`
	if got, ok := ResolveRef([]byte(tree), "@e2"); !ok || got != "@e2" {
		t.Errorf("exact: %q %v", got, ok)
	}
	// @e9 is gone; fuzzy should land on the stable Button, not fail.
	if got, ok := ResolveRef([]byte(tree), "@e9"); !ok || got != "@e2" {
		t.Errorf("fuzzy: %q %v", got, ok)
	}
	// Empty tree: honest failure, never a hallucinated ref.
	if _, ok := ResolveRef([]byte(`{"nodes":[]}`), "@e5"); ok {
		t.Error("empty tree should not resolve")
	}
}

func TestReplayPassAndAbort(t *testing.T) {
	shot := solidPNG(t, 4, 4, color.RGBA{1, 2, 3, 255})
	rec := &Recording{Name: "login", Steps: []Step{
		{Action: "tap", Ref: "@e2"},
		{Action: "type", Text: "hello"},
	}}
	f := &fakeRunner{tree: `{"nodes":[{"ref":"@e2","class":"Button","text":"Send"}]}`, shot: shot}
	rep := Replay(rec, f, ReplayOptions{DiffThreshold: 0, Fuzzy: true})
	if rep.Failed || len(rep.Steps) != 2 {
		t.Fatalf("replay: %+v", rep)
	}
	if len(f.calls) != 2 || f.calls[0] != "tap @e2" || f.calls[1] != "type hello" {
		t.Errorf("calls: %v", f.calls)
	}

	// Unknown action aborts with the step marked.
	bad := &Recording{Name: "bad", Steps: []Step{{Action: "dance"}}}
	rep = Replay(bad, f, ReplayOptions{})
	if !rep.Failed || !strings.Contains(rep.Steps[0].Err, "unknown action") {
		t.Errorf("bad action: %+v", rep)
	}
}

func TestReplayVisualGate(t *testing.T) {
	want := solidPNG(t, 4, 4, color.RGBA{0, 0, 0, 255})
	gotShot := solidPNG(t, 4, 4, color.RGBA{255, 255, 255, 255})
	rec := &Recording{Name: "v", Steps: []Step{{Action: "tap", Ref: "@e1", Shot: string(want)}}}
	f := &fakeRunner{tree: `{"nodes":[{"ref":"@e1","class":"Button"}]}`, shot: gotShot}
	rep := Replay(rec, f, ReplayOptions{DiffThreshold: 0.05})
	if !rep.Failed || !strings.Contains(rep.Steps[0].Err, "visual diff") {
		t.Errorf("visual gate: %+v", rep)
	}
	if rep.Steps[0].DiffPct < 0.99 {
		t.Errorf("diff pct: %v", rep.Steps[0].DiffPct)
	}
}

func TestSaveLoadRoundtrip(t *testing.T) {
	rec := &Recording{Name: "r", Steps: []Step{{Action: "tap", Ref: "@e1"}}}
	p := filepath.Join(t.TempDir(), "r.json")
	if err := rec.Save(p); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(p)
	if !strings.Contains(string(raw), `"version": 1`) {
		t.Errorf("version missing: %s", raw)
	}
	loaded, err := Load(p)
	if err != nil || loaded.Name != "r" || loaded.Steps[0].Ref != "@e1" {
		t.Errorf("roundtrip: %+v %v", loaded, err)
	}
	if _, err := Load(filepath.Join(t.TempDir(), "nope.json")); err == nil {
		t.Error("missing file should error")
	}
}
