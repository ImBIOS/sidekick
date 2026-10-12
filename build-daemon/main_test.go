// SPDX-License-Identifier: FSL-1.1-ALv2
// Copyright 2026 ImBIOS — licensed under FSL-1.1-ALv2, see LICENSE.md.
package main

import (
	"fmt"
	"strings"
	"testing"
)

// fakeRunner pretends to be xcodebuild: emits canned lines, exits per fail.
type fakeRunner struct {
	lines []string
	fail  bool
}

func (f fakeRunner) Run(workdir, scheme, ref string, emit func(string)) error {
	if workdir == "" || scheme == "" {
		return fmt.Errorf("workdir and scheme are required")
	}
	emit("$ fake xcodebuild " + scheme + " ref=" + ref)
	for _, l := range f.lines {
		emit(l)
	}
	if f.fail {
		return fmt.Errorf("exit status 65")
	}
	return nil
}

// memPoster records what driveBuild would stream to the control-plane.
type memPoster struct {
	logs   map[string][]string
	status map[string]string
}

func newMemPoster() *memPoster {
	return &memPoster{logs: map[string][]string{}, status: map[string]string{}}
}

func (m *memPoster) PostBuildLogs(id string, lines []string) error {
	m.logs[id] = append(m.logs[id], lines...)
	return nil
}

func (m *memPoster) FinishBuild(id, status string) error {
	m.status[id] = status
	return nil
}

func TestDriveBuildSuccess(t *testing.T) {
	p := newMemPoster()
	build := map[string]any{"build_id": "bld_1", "scheme": "MyApp", "ref": "abc123"}
	driveBuild(p, build, fakeRunner{lines: []string{"compile A", "link B"}}, "/tmp/proj")
	if p.status["bld_1"] != "succeeded" {
		t.Errorf("status = %q, want succeeded", p.status["bld_1"])
	}
	joined := strings.Join(p.logs["bld_1"], "\n")
	for _, want := range []string{"compile A", "link B", "build succeeded", "ref=abc123"} {
		if !strings.Contains(joined, want) {
			t.Errorf("logs missing %q:\n%s", want, joined)
		}
	}
}

func TestDriveBuildFailure(t *testing.T) {
	p := newMemPoster()
	build := map[string]any{"build_id": "bld_2", "scheme": "MyApp"}
	driveBuild(p, build, fakeRunner{fail: true}, "/tmp/proj")
	if p.status["bld_2"] != "failed" {
		t.Errorf("status = %q, want failed", p.status["bld_2"])
	}
	if !strings.Contains(strings.Join(p.logs["bld_2"], "\n"), "build failed") {
		t.Errorf("missing failure line: %q", p.logs["bld_2"])
	}
}
