// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 ImBIOS — licensed under Apache-2.0, see LICENSE-APACHE.md.
package testrun

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Plain-English testing: one CLI arg per step, grounded against the live
// tree and executed over adb. This is deliberately NOT an LLM call — the
// agent (Claude, Cursor, Codex) already reasons; Sidekick grounds each
// English verb to a concrete element deterministically, so runs are
// reproducible and failures point at the exact step and candidates.
//
// Grammar (verb first, case-insensitive):
//
//	tap|click <target>            tap the element named target
//	type <text> [into <field>]    type, optionally tapping the field first
//	swipe <up|down|left|right>
//	press <back|home|enter>
//	wait <1s|500ms|2>
//	expect <text> [visible|gone]  assert tree contains (or lacks) text
//
// Targets match element text or content-desc, exact first, then contains.
type NLStep struct {
	Raw    string
	Action string // tap|type|swipe|press|wait|expect
	Target string // tap target / expect text
	Text   string // type text
	Field  string // type-into field
	Dir    string
	Key    string
	Wait   time.Duration
	Gone   bool // expect ... gone
}

// ParseNLStep parses one English step.
func ParseNLStep(raw string) (NLStep, error) {
	s := NLStep{Raw: raw}
	fields := strings.Fields(raw)
	if len(fields) == 0 {
		return s, fmt.Errorf("empty step")
	}
	verb := strings.ToLower(fields[0])
	rest := strings.TrimSpace(raw[len(fields[0]):])
	switch verb {
	case "tap", "click":
		if rest == "" {
			return s, fmt.Errorf("tap what?")
		}
		s.Action, s.Target = "tap", rest
	case "type":
		if rest == "" {
			return s, fmt.Errorf("type what?")
		}
		s.Action = "type"
		if i := strings.LastIndex(strings.ToLower(rest), " into "); i >= 0 {
			s.Text = strings.TrimSpace(rest[:i])
			s.Field = strings.TrimSpace(rest[i+len(" into "):])
		} else {
			s.Text = rest
		}
	case "swipe":
		d := strings.ToLower(strings.TrimSpace(rest))
		switch d {
		case "up", "down", "left", "right":
			s.Action, s.Dir = "swipe", d
		default:
			return s, fmt.Errorf("swipe where? up|down|left|right")
		}
	case "press":
		k := strings.ToLower(strings.TrimSpace(rest))
		switch k {
		case "back", "home", "enter":
			s.Action, s.Key = "press", k
		default:
			return s, fmt.Errorf("press what? back|home|enter")
		}
	case "wait":
		d, err := parseWait(strings.TrimSpace(rest))
		if err != nil {
			return s, err
		}
		s.Action, s.Wait = "wait", d
	case "expect":
		low := strings.ToLower(rest)
		s.Action = "expect"
		switch {
		case strings.HasSuffix(low, " gone") || strings.HasSuffix(low, " absent"):
			s.Gone = true
			s.Target = strings.TrimSpace(rest[:len(rest)-len(" gone")])
			if strings.HasSuffix(low, " absent") {
				s.Target = strings.TrimSpace(rest[:len(rest)-len(" absent")])
			}
		case strings.HasSuffix(low, " visible"):
			s.Target = strings.TrimSpace(rest[:len(rest)-len(" visible")])
		default:
			s.Target = rest
		}
		if s.Target == "" {
			return s, fmt.Errorf("expect what?")
		}
	default:
		return s, fmt.Errorf("unknown verb %q (tap|type|swipe|press|wait|expect)", fields[0])
	}
	return s, nil
}

func parseWait(s string) (time.Duration, error) {
	if s == "" {
		return 0, fmt.Errorf("wait how long? e.g. wait 2s")
	}
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}
	if n, err := strconv.Atoi(s); err == nil {
		return time.Duration(n) * time.Second, nil
	}
	return 0, fmt.Errorf("bad duration %q", s)
}

// FindInTree resolves a human target to an AX ref: exact text/desc match
// first, then contains, preferring clickable elements and shorter labels.
// Returns ok=false with candidates for the error message.
func FindInTree(tree []byte, target string) (ref string, ok bool, candidates []string) {
	nodes := collectNodes(tree)
	if len(nodes) == 0 {
		return "", false, nil
	}
	want := strings.ToLower(target)
	best, bestScore := "", -1000
	for _, n := range nodes {
		label := n.Text
		if label == "" {
			label = n.Desc
		}
		if label == "" || n.Ref == "" {
			continue
		}
		if len(candidates) < 8 {
			candidates = append(candidates, label)
		}
		lower := strings.ToLower(label)
		if lower == want {
			// Exact hit wins outright (first clickable exact preferred).
			if n.Clickable {
				return n.Ref, true, nil
			}
			if bestScore < 500 {
				bestScore, best = 500, n.Ref
			}
			continue
		}
		if !strings.Contains(lower, want) && !strings.Contains(want, lower) {
			continue
		}
		score := 10 - len(label)/20
		if n.Clickable {
			score += 10
		}
		if score > bestScore {
			bestScore, best = score, n.Ref
		}
	}
	if best == "" {
		return "", false, candidates
	}
	return best, true, nil
}

// RunNL executes grounded English steps, stopping at the first failure
// with the candidates listed (reproducible, debuggable).
func RunNL(steps []NLStep, run Runner) *Report {
	rep := &Report{Name: "nl"}
	for i, s := range steps {
		sr := StepResult{Step: i + 1, Action: "nl:" + s.Action, Ref: s.Target}
		sr.Resolved, sr.Err = runNLStep(run, s)
		rep.Steps = append(rep.Steps, sr)
		if sr.Err != "" {
			rep.Failed = true
			break
		}
	}
	return rep
}

func runNLStep(run Runner, s NLStep) (resolved, errStr string) {
	treeOf := func() ([]byte, string) {
		tree, err := run.Tree()
		if err != nil {
			return nil, "read tree: " + err.Error()
		}
		return tree, ""
	}
	switch s.Action {
	case "tap":
		tree, e := treeOf()
		if e != "" {
			return "", e
		}
		ref, ok, cands := FindInTree(tree, s.Target)
		if !ok {
			return "", fmt.Sprintf("no element %q (saw: %s)", s.Target, strings.Join(cands, ", "))
		}
		if err := run.Tap(ref); err != nil {
			return ref, err.Error()
		}
		return ref, ""
	case "type":
		if s.Field != "" {
			tree, e := treeOf()
			if e != "" {
				return "", e
			}
			ref, ok, cands := FindInTree(tree, s.Field)
			if !ok {
				return "", fmt.Sprintf("no field %q (saw: %s)", s.Field, strings.Join(cands, ", "))
			}
			resolved = ref
			if err := run.Tap(ref); err != nil {
				return resolved, err.Error()
			}
		}
		if err := run.Type(s.Text); err != nil {
			return resolved, err.Error()
		}
		return resolved, ""
	case "swipe":
		if err := run.Swipe(s.Dir); err != nil {
			return "", err.Error()
		}
		return "", ""
	case "press":
		if err := run.Press(s.Key); err != nil {
			return "", err.Error()
		}
		return "", ""
	case "wait":
		time.Sleep(s.Wait)
		return "", ""
	case "expect":
		tree, e := treeOf()
		if e != "" {
			return "", e
		}
		found := treeContains(tree, s.Target)
		if s.Gone && found {
			return "", fmt.Sprintf("expected %q gone, still visible", s.Target)
		}
		if !s.Gone && !found {
			return "", fmt.Sprintf("expected %q visible, not found", s.Target)
		}
		return "", ""
	default:
		return "", "unknown action: " + s.Action
	}
}

// treeContains reports whether any node text/desc contains target.
func treeContains(tree []byte, target string) bool {
	want := strings.ToLower(target)
	for _, n := range collectNodes(tree) {
		if strings.Contains(strings.ToLower(n.Text), want) || strings.Contains(strings.ToLower(n.Desc), want) {
			return true
		}
	}
	return false
}
func collectNodes(tree []byte) []treeNodeFlat {
	var doc struct {
		Nodes []treeNode `json:"nodes"`
		Root  *treeNode  `json:"root"`
	}
	if err := json.Unmarshal(tree, &doc); err != nil {
		return nil
	}
	var out []treeNodeFlat
	var walk func(n *treeNode)
	walk = func(n *treeNode) {
		out = append(out, treeNodeFlat{Ref: n.Ref, Class: n.Class, Text: n.Text, Desc: n.Desc, Clickable: n.Clickable})
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
	return out
}

type treeNodeFlat struct {
	Ref       string
	Class     string
	Text      string
	Desc      string
	Clickable bool
}
