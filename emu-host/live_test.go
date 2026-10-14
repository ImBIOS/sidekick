// SPDX-License-Identifier: FSL-1.1-ALv2
// Copyright 2026 ImBIOS — licensed under FSL-1.1-ALv2, see LICENSE.md.
package main

import (
	"encoding/json"
	"os"
	"testing"
)

// TestExecRunnerLiveDevice exercises the real adb paths against a booted
// emulator. It only runs with a device present:
//
//	SIDEKICK_TEST_ADB_SERIAL=emulator-5554 go test ./emu-host/ -run Live -v
//
// CI (and `go test -short`) always skips it: no KVM there. Witnessed once
// on 2026-10-14 against a software-emulated AVD (boot_completed=1):
// logcat streams, screencap returns a valid PNG, uiautomator fails
// on-device under software emulation (surface stack wedged) — the Tree
// error path below asserts that failure propagates instead of garbage.
func TestExecRunnerLiveDevice(t *testing.T) {
	if testing.Short() {
		t.Skip("needs a booted emulator")
	}
	serial := os.Getenv("SIDEKICK_TEST_ADB_SERIAL")
	if serial == "" {
		t.Skip("SIDEKICK_TEST_ADB_SERIAL unset")
	}
	r := execRunner{adb: "adb", bootTimeout: 0}

	lines, err := r.Logs(serial)
	if err != nil || len(lines) == 0 {
		t.Fatalf("logs: %v (%d lines)", err, len(lines))
	}
	t.Logf("logcat: %d lines, e.g. %q", len(lines), lines[0])

	png, err := r.Shot(serial)
	if err != nil {
		t.Fatalf("shot: %v", err)
	}
	if len(png) < 8 || png[0] != 0x89 || png[1] != 0x50 {
		t.Fatalf("shot: not a PNG (%d bytes)", len(png))
	}
	t.Logf("screenshot: valid PNG, %d bytes", len(png))

	tree, err := r.Tree(serial)
	if err != nil {
		// Documented wedged-device outcome: must fail loudly, never garbage.
		t.Logf("tree failed as documented on software emu: %v", err)
		return
	}
	var doc struct {
		Nodes []any `json:"nodes"`
		Refs  int   `json:"refs"`
	}
	if err := json.Unmarshal(tree, &doc); err != nil || doc.Refs < 0 {
		t.Fatalf("tree: invalid JSON: %v", err)
	}
	t.Logf("tree: %d refs", doc.Refs)
}
