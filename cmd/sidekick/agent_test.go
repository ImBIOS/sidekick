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
