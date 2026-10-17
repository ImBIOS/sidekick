// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 ImBIOS — licensed under Apache-2.0, see LICENSE-APACHE.md.

// Package testrun is the record → replay → verify core behind
// `sidekick test`. A recording is plain JSON: UI verbs with the AX refs
// and screenshots captured at record time. Replay re-resolves every ref
// against a fresh tree (fuzzy when the UI moved), drives adb input, and
// diffs screenshots for visual regression. Pure logic, no devices —
// the adb transport is a Runner interface faked in tests.
package testrun

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/png"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
)

// Step is one recorded verb.
type Step struct {
	Action string `json:"action"` // tap | type | swipe | press | wait
	Ref    string `json:"ref,omitempty"`
	Text   string `json:"text,omitempty"`
	Dir    string `json:"dir,omitempty"`
	Key    string `json:"key,omitempty"`
	Ms     int    `json:"ms,omitempty"`
	Shot   string `json:"shot,omitempty"` // expected screenshot (PNG bytes, base64) for visual check
}

// Recording is a versioned test: replay + verify.
type Recording struct {
	Version int    `json:"version"`
	Name    string `json:"name"`
	Steps   []Step `json:"steps"`
}

// Load reads a recording file.
func Load(path string) (*Recording, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var r Recording
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("bad recording: %w", err)
	}
	if r.Version != 1 {
		return nil, fmt.Errorf("unsupported recording version %d", r.Version)
	}
	if len(r.Steps) == 0 {
		return nil, fmt.Errorf("recording has no steps")
	}
	return &r, nil
}

// Save writes a recording file.
func (r *Recording) Save(path string) error {
	if r.Version == 0 {
		r.Version = 1
	}
	raw, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}

// Runner executes device primitives. execADB shells adb; fakes in tests.
type Runner interface {
	Tap(ref string) error
	Type(text string) error
	Swipe(dir string) error
	Press(key string) error
	Screenshot() ([]byte, error)
	Tree() ([]byte, error)
}

// Report is one replay result, CI-friendly (exit code derives from Failed).
type Report struct {
	Name   string       `json:"name"`
	Steps  []StepResult `json:"steps"`
	Failed bool         `json:"failed"`
}

// StepResult records what happened per step.
type StepResult struct {
	Step     int     `json:"step"`
	Action   string  `json:"action"`
	Ref      string  `json:"ref,omitempty"`
	Resolved string  `json:"resolved,omitempty"`
	DiffPct  float64 `json:"diff_pct,omitempty"`
	Err      string  `json:"err,omitempty"`
}

// ReplayOptions tunes verification strictness.
type ReplayOptions struct {
	// DiffThreshold fails a step whose screenshot differs more than this
	// fraction (0..1). 0 disables visual checks.
	DiffThreshold float64
	// Fuzzy allows stale refs to re-resolve by text/class/bounds proximity.
	Fuzzy bool
}

// Replay drives every step, resolving refs against a fresh tree first.
// A failed step aborts the run (like a real test runner).
func Replay(r *Recording, run Runner, opts ReplayOptions) *Report {
	rep := &Report{Name: r.Name}
	for i, s := range r.Steps {
		sr := StepResult{Step: i + 1, Action: s.Action, Ref: s.Ref}
		ref := s.Ref
		if ref != "" && opts.Fuzzy {
			if tree, err := run.Tree(); err == nil {
				if resolved, ok := ResolveRef(tree, ref); ok && resolved != ref {
					ref = resolved
					sr.Resolved = resolved
				}
			}
		}
		sr.Err = runStep(run, s, ref)
		if s.Shot != "" && sr.Err == "" && opts.DiffThreshold > 0 {
			got, err := run.Screenshot()
			if err != nil {
				sr.Err = "screenshot: " + err.Error()
			} else if pct, derr := DiffPNG([]byte(s.Shot), got); derr != nil {
				sr.Err = "diff: " + derr.Error()
			} else {
				sr.DiffPct = pct
				if pct > opts.DiffThreshold {
					sr.Err = fmt.Sprintf("visual diff %.1f%% exceeds %.1f%%", pct*100, opts.DiffThreshold*100)
				}
			}
		}
		rep.Steps = append(rep.Steps, sr)
		if sr.Err != "" {
			rep.Failed = true
			break
		}
	}
	return rep
}

func runStep(run Runner, s Step, ref string) string {
	var err error
	switch s.Action {
	case "tap":
		err = run.Tap(ref)
	case "type":
		err = run.Type(s.Text)
	case "swipe":
		err = run.Swipe(s.Dir)
	case "press":
		err = run.Press(s.Key)
	case "wait":
		if s.Ms > 0 {
			time.Sleep(time.Duration(s.Ms) * time.Millisecond)
		}
		return ""
	case "shot":
		return "" // screenshot is captured by the visual-check block below
	default:
		return "unknown action: " + s.Action
	}
	if err != nil {
		return err.Error()
	}
	return ""
}

// --- fuzzy ref resolution ---

// treeNode is the subset of the AX tree we match on.
type treeNode struct {
	Ref       string     `json:"ref"`
	Class     string     `json:"class"`
	Text      string     `json:"text"`
	Desc      string     `json:"desc"`
	Bounds    string     `json:"bounds"`
	Clickable bool       `json:"clickable"`
	Children  []treeNode `json:"children"`
}

// ResolveRef finds ref in tree; if the exact ref is gone, returns the
// closest element by text/class/bounds similarity (ok=false if nothing
// scores above the floor). This is the "agent-fix on moved elements".
func ResolveRef(tree []byte, ref string) (string, bool) {
	var doc struct {
		Nodes []treeNode `json:"nodes"`
		Root  *treeNode  `json:"root"`
	}
	if err := json.Unmarshal(tree, &doc); err != nil {
		return ref, false
	}
	var all []*treeNode
	var walk func(n *treeNode)
	walk = func(n *treeNode) {
		all = append(all, n)
		for i := range n.Children {
			walk(&n.Children[i])
		}
	}
	for i := range doc.Nodes {
		walk(&doc.Nodes[i])
	}
	if doc.Root != nil {
		walk(doc.Root)
	}
	for _, n := range all {
		if n.Ref == ref {
			return ref, true
		}
	}
	// Exact ref missing: match by the stale element's own attributes is
	// impossible (we only know the ref string), so score candidates by
	// structural stability: prefer clickable leaf nodes with text, in
	// document order near the ref's numeric position.
	idx := refIndex(ref)
	best, bestScore := "", -1.0
	for _, n := range all {
		score := 0.0
		if n.Text != "" || n.Desc != "" {
			score += 2
		}
		if n.Class == "Button" {
			score += 1
		}
		score -= math.Abs(float64(numericRef(n.Ref)-idx)) * 0.1
		if score > bestScore {
			bestScore, best = score, n.Ref
		}
	}
	if best == "" || bestScore < 1.0 {
		return ref, false
	}
	return best, true
}

func refIndex(ref string) int {
	n, _ := strconv.Atoi(strings.TrimPrefix(ref, "@e"))
	return n
}

func numericRef(ref string) int { return refIndex(ref) }

// --- visual diff ---

// DiffPNG returns the fraction of differing pixels (0 = identical).
// Images must decode as PNG; different sizes compare over the overlap and
// count the overhang as fully different.
func DiffPNG(a, b []byte) (float64, error) {
	ia, err := png.Decode(bytes.NewReader(a))
	if err != nil {
		return 0, fmt.Errorf("expected shot: %w", err)
	}
	ib, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		return 0, fmt.Errorf("actual shot: %w", err)
	}
	ba, bb := ia.Bounds(), ib.Bounds()
	w := max(bb.Max.X, ba.Max.X)
	h := max(bb.Max.Y, ba.Max.Y)
	ow := min(bb.Max.X, ba.Max.X)
	oh := min(bb.Max.Y, ba.Max.Y)
	var diff, total int64
	total = int64(w) * int64(h)
	for y := 0; y < oh; y++ {
		for x := 0; x < ow; x++ {
			ar, ag, ab, aa := ia.At(x, y).RGBA()
			br, bg, bb2, ba2 := ib.At(x, y).RGBA()
			if ar != br || ag != bg || ab != bb2 || aa != ba2 {
				diff++
			}
		}
	}
	diff += int64(w*h - ow*oh) // overhang counts as different
	if total == 0 {
		return 0, fmt.Errorf("empty images")
	}
	return float64(diff) / float64(total), nil
}
