// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 ImBIOS — licensed under Apache-2.0, see LICENSE-APACHE.md.
package testrun

import (
	"strings"
	"testing"
	"time"
)

func TestParseNLStep(t *testing.T) {
	cases := []struct {
		in     string
		action string
		target string
		text   string
		field  string
		dir    string
		gone   bool
	}{
		{"tap Send", "tap", "Send", "", "", "", false},
		{"TAP the send button", "tap", "the send button", "", "", "", false},
		{`type hello into message`, "type", "", "hello", "message", "", false},
		{"type hi", "type", "", "hi", "", "", false},
		{"swipe up", "swipe", "", "", "", "up", false},
		{"press back", "press", "", "", "", "", false},
		{"wait 2s", "wait", "", "", "", "", false},
		{"expect Welcome visible", "expect", "Welcome", "", "", "", false},
		{"expect spinner gone", "expect", "spinner", "", "", "", true},
	}
	for _, tc := range cases {
		s, err := ParseNLStep(tc.in)
		if err != nil {
			t.Errorf("%q: %v", tc.in, err)
			continue
		}
		if s.Action != tc.action || s.Target != tc.target || s.Text != tc.text ||
			s.Field != tc.field || s.Dir != tc.dir || s.Gone != tc.gone {
			t.Errorf("%q: %+v", tc.in, s)
		}
	}
	bad := []string{"", "dance now", "tap", "type", "swipe sideways", "press power", "wait", "wait soon", "expect"}
	for _, in := range bad {
		if _, err := ParseNLStep(in); err == nil {
			t.Errorf("%q should fail", in)
		}
	}
	if s, _ := ParseNLStep("wait 2"); s.Wait != 2*time.Second {
		t.Errorf("bare wait: %v", s.Wait)
	}
}

func TestFindInTree(t *testing.T) {
	tree := []byte(`{"nodes":[
		{"ref":"@e1","class":"FrameLayout","children":[
			{"ref":"@e2","class":"Button","text":"Send","clickable":true},
			{"ref":"@e3","class":"TextView","text":"Send necesse","clickable":false},
			{"ref":"@e4","class":"Button","text":"Cancel","clickable":true}]}]}`)
	// Exact prefers the clickable exact hit.
	if ref, ok, _ := FindInTree(tree, "Send"); !ok || ref != "@e2" {
		t.Errorf("exact: %q %v", ref, ok)
	}
	// Case-insensitive contains.
	if ref, ok, _ := FindInTree(tree, "cancel"); !ok || ref != "@e4" {
		t.Errorf("contains: %q %v", ref, ok)
	}
	// Desc fallback.
	tree2 := []byte(`{"nodes":[{"ref":"@e9","class":"ImageView","desc":"Search"}]}`)
	if ref, ok, _ := FindInTree(tree2, "search"); !ok || ref != "@e9" {
		t.Errorf("desc: %q %v", ref, ok)
	}
	// Miss lists candidates, never hallucinates.
	if ref, ok, cands := FindInTree(tree, "Zebra"); ok || ref != "" || len(cands) == 0 {
		t.Errorf("miss: %q %v %v", ref, ok, cands)
	}
	if _, ok, _ := FindInTree([]byte(`{"nodes":[]}`), "x"); ok {
		t.Error("empty tree resolves")
	}
	if _, ok, _ := FindInTree([]byte(`{{{`), "x"); ok {
		t.Error("garbage tree resolves")
	}
}

func TestRunNL(t *testing.T) {
	tree := `{"nodes":[{"ref":"@e1","class":"Button","text":"Send","clickable":true},{"ref":"@e2","class":"TextView","text":"Hello"}]}`
	f := &fakeRunner{tree: tree, shot: []byte("png")}
	steps := []NLStep{
		{Action: "tap", Target: "send"},
		{Action: "type", Text: "hi"},
		{Action: "expect", Target: "hello"},
		{Action: "expect", Target: "spinner", Gone: true},
	}
	rep := RunNL(mustParseAll(t, []string{"tap send", "type hi", "expect Hello", "expect spinner gone"}), f)
	_ = steps
	if rep.Failed || len(rep.Steps) != 4 {
		t.Fatalf("run: %+v", rep)
	}
	if rep.Steps[0].Resolved != "@e1" {
		t.Errorf("resolved: %q", rep.Steps[0].Resolved)
	}
	if len(f.calls) != 2 || f.calls[0] != "tap @e1" || f.calls[1] != "type hi" {
		t.Errorf("calls: %v", f.calls)
	}
	// Miss aborts with candidates.
	rep = RunNL(mustParseAll(t, []string{"tap Zebra"}), f)
	if !rep.Failed || !strings.Contains(rep.Steps[0].Err, "saw:") {
		t.Errorf("miss should abort with candidates: %+v", rep)
	}
	// Failed assert aborts.
	rep = RunNL(mustParseAll(t, []string{"expect Goodbye"}), f)
	if !rep.Failed || !strings.Contains(rep.Steps[0].Err, "not found") {
		t.Errorf("assert: %+v", rep)
	}
}

func mustParseAll(t *testing.T, ins []string) []NLStep {
	t.Helper()
	var out []NLStep
	for _, in := range ins {
		s, err := ParseNLStep(in)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, s)
	}
	return out
}
