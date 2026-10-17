// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 ImBIOS — licensed under Apache-2.0, see LICENSE-APACHE.md.
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The embedded skill copy must never drift from the source of truth:
// `cp skill/SKILL.md cmd/sidekick/skill-embed/SKILL.md` after editing.
func TestSkillEmbedMatchesSource(t *testing.T) {
	embedded, err := skillFS.ReadFile("skill-embed/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile(filepath.Join("..", "..", "skill", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(embedded) != string(src) {
		t.Error("skill-embed/SKILL.md drifted from skill/SKILL.md; copy it over")
	}
}

func TestAppendNudgeIdempotent(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "AGENTS.md")
	if err := appendNudge(p); err != nil {
		t.Fatal(err)
	}
	if err := appendNudge(p); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(p)
	if strings.Count(string(raw), "## Sidekick") != 1 {
		t.Errorf("nudge appended %d times, want 1", strings.Count(string(raw), "## Sidekick"))
	}
}

func TestAgentSetupInstalls(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	proj := t.TempDir()
	cmd := agentSetupCmd()
	cmd.SetArgs([]string{"--dir", proj})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(home, ".claude", "skills", "sidekick", "SKILL.md"))
	if err != nil || !strings.Contains(string(raw), "sidekick ios create") {
		t.Errorf("home skill missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(proj, ".claude", "skills", "sidekick", "SKILL.md")); err != nil {
		t.Errorf("project skill missing: %v", err)
	}
	agents, err := os.ReadFile(filepath.Join(proj, "AGENTS.md"))
	if err != nil || !strings.Contains(string(agents), "## Sidekick") {
		t.Errorf("AGENTS.md nudge missing: %v", err)
	}
}

func TestRefCenterClickableAncestor(t *testing.T) {
	tree := []byte(`{"nodes":[{"ref":"@e1","class":"FrameLayout","bounds":"[0,0][1080,100]","clickable":true,"children":[{"ref":"@e2","class":"TextView","text":"Gmail","bounds":"[309,1487][519,1759]"}]}]}`)
	// @e2 itself is not clickable: expect the clickable parent's center.
	x, y, err := refCenter(tree, "@e2")
	if err != nil {
		t.Fatal(err)
	}
	if x != 540 || y != 50 {
		t.Errorf("got (%d,%d), want parent center (540,50)", x, y)
	}
	// Exact clickable node resolves to itself.
	tree2 := []byte(`{"nodes":[{"ref":"@e5","class":"Button","bounds":"[100,200][400,300]","clickable":true}]}`)
	x, y, err = refCenter(tree2, "@e5")
	if err != nil || x != 250 || y != 250 {
		t.Errorf("got (%d,%d) %v, want (250,250)", x, y, err)
	}
	if _, _, err := refCenter(tree2, "@e9"); err == nil {
		t.Error("missing ref should error")
	}
}

func TestParseFoldState(t *testing.T) {
	out := "Supported states: [\n  DeviceState{identifier=1, name='CLOSED'},\n  DeviceState{identifier=2, name='HALF_OPENED'},\n  DeviceState{identifier=3, name='OPENED'},\n]"
	if got := parseFoldState(out, "OPENED"); got != "3" {
		t.Errorf("open: %q", got)
	}
	if got := parseFoldState(out, "HALF_OPENED"); got != "2" {
		t.Errorf("half: %q", got)
	}
	if got := parseFoldState(out, "TABLET"); got != "" {
		t.Errorf("missing: %q", got)
	}
}
