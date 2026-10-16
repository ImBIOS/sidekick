// SPDX-License-Identifier: FSL-1.1-ALv2
// Copyright 2026 ImBIOS — licensed under FSL-1.1-ALv2, see LICENSE.md.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

const sampleDump = `<?xml version="1.0" encoding="UTF-8"?>
<hierarchy rotation="0">
  <node index="0" text="" resource-id="" class="android.widget.FrameLayout" package="com.app" content-desc="" checkable="false" checked="false" clickable="false" enabled="true" focusable="false" focused="false" scrollable="false" long-clickable="false" password="false" selected="false" bounds="[0,0][1080,2400]">
    <node index="0" text="Send" resource-id="com.app:id/send" class="android.widget.Button" package="com.app" content-desc="Send message" checkable="false" checked="false" clickable="true" enabled="true" focusable="true" focused="false" scrollable="false" long-clickable="false" password="false" selected="false" bounds="[100,200][400,300]" />
    <node index="1" text="" resource-id="" class="android.view.View" package="com.app" content-desc="" checkable="false" checked="false" clickable="false" enabled="true" focusable="false" focused="false" scrollable="false" long-clickable="false" password="false" selected="false" bounds="[0,0][0,0]" />
  </node>
</hierarchy>`

func TestUiaToJSON(t *testing.T) {
	raw, err := uiaToJSON([]byte(sampleDump))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Nodes []struct {
			Ref       string `json:"ref"`
			Class     string `json:"class"`
			Text      string `json:"text"`
			Desc      string `json:"desc"`
			Clickable bool   `json:"clickable"`
			Children  []struct {
				Ref       string `json:"ref"`
				Class     string `json:"class"`
				Text      string `json:"text"`
				Clickable bool   `json:"clickable"`
			} `json:"children"`
		} `json:"nodes"`
		Refs int `json:"refs"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Refs != 3 || len(doc.Nodes) != 1 {
		t.Fatalf("refs=%d nodes=%d", doc.Refs, len(doc.Nodes))
	}
	root := doc.Nodes[0]
	if root.Ref != "@e1" || root.Class != "FrameLayout" {
		t.Errorf("root: %+v", root)
	}
	btn := root.Children[0]
	if btn.Ref != "@e2" || btn.Class != "Button" || btn.Text != "Send" {
		t.Errorf("button: %+v", btn)
	}
	if !root.Children[0].Clickable {
		t.Error("button should be clickable")
	}
	if root.Children[1].Ref != "@e3" {
		t.Errorf("third ref: %+v", root.Children[1])
	}
}

func TestUiaToJSONBadInput(t *testing.T) {
	if _, err := uiaToJSON([]byte("not xml")); err == nil {
		t.Error("expected error for garbage")
	}
	raw, err := uiaToJSON([]byte(`<hierarchy></hierarchy>`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"refs":0`) {
		t.Errorf("empty hierarchy: %s", raw)
	}
}

// fakeRunner pretends to be emulator+adb.
type fakeRunner struct {
	serial string
	tree   []byte
	shot   []byte
	logs   []string
	boots  []string
	kills  []string
}

func (f *fakeRunner) Boot(profile string) (string, error) {
	if profile == "" {
		return "", errors.New("profile required")
	}
	f.boots = append(f.boots, profile)
	return f.serial, nil
}
func (f *fakeRunner) Tree(serial string) ([]byte, error)   { return f.tree, nil }
func (f *fakeRunner) Shot(serial string) ([]byte, error)   { return f.shot, nil }
func (f *fakeRunner) Logs(serial string) ([]string, error) { return f.logs, nil }
func (f *fakeRunner) Destroy(serial string) error {
	f.kills = append(f.kills, serial)
	return nil
}

func (f *fakeRunner) StartVideo(serial string) error { return nil }
func (f *fakeRunner) StopVideo(serial string) ([]byte, error) {
	return []byte("mp4bytes"), nil
}
func (f *fakeRunner) StartNetcap(serial string) error { return nil }
func (f *fakeRunner) StopNetcap(serial string) ([]byte, error) {
	return []byte("pcapbytes"), nil
}
func (f *fakeRunner) Rotate(serial string, degrees int) error {
	if degrees != 0 && degrees != 90 && degrees != 180 && degrees != 270 {
		return fmt.Errorf("degrees must be one of 0, 90, 180, 270")
	}
	return nil
}
func (f *fakeRunner) SetPosture(serial, posture string) error {
	if posture != "open" && posture != "half-open" && posture != "closed" {
		return fmt.Errorf("bad posture")
	}
	return nil
}

// memPoster records control-plane posts; notFoundIDs simulate deleted emus.
type memPoster struct {
	ready       map[string]string
	trees       map[string]int
	shots       map[string]int
	logs        map[string]int
	notFoundIDs map[string]bool
}

func newMemPoster() *memPoster {
	return &memPoster{ready: map[string]string{}, trees: map[string]int{},
		shots: map[string]int{}, logs: map[string]int{}, notFoundIDs: map[string]bool{}}
}

func (m *memPoster) EmuReady(id, host, target string) error {
	m.ready[id] = host + "|" + target
	return nil
}
func (m *memPoster) PutEmuTree(id string, tree []byte) error {
	if m.notFoundIDs[id] {
		return errors.New("sidekick: HTTP 404: emu not found")
	}
	m.trees[id] = len(tree)
	return nil
}
func (m *memPoster) PutEmuShot(id string, png []byte) error {
	m.shots[id] = len(png)
	return nil
}
func (m *memPoster) PostEmuLogs(id string, lines []string) error {
	m.logs[id] += len(lines)
	return nil
}

func TestProvisionPublishes(t *testing.T) {
	r := &fakeRunner{serial: "emulator-5554", tree: []byte(`{"nodes":[]}`), shot: []byte("png"), logs: []string{"boot complete"}}
	p := newMemPoster()
	h := &host{fc: p, pool: "linux-1", adbAddr: "10.0.0.9", adbPort: "5555", runner: r, serials: map[string]string{}}
	h.provision("emu_1", "Pixel_7_API_34")
	if p.ready["emu_1"] != "linux-1|10.0.0.9:5555" {
		t.Errorf("ready: %q", p.ready["emu_1"])
	}
	if r.boots[0] != "Pixel_7_API_34" {
		t.Errorf("boot profile: %v", r.boots)
	}
	if p.trees["emu_1"] == 0 || p.shots["emu_1"] != 3 || p.logs["emu_1"] != 1 {
		t.Errorf("initial state: %+v", p)
	}
	if h.serials["emu_1"] != "emulator-5554" {
		t.Errorf("serial not tracked")
	}
}

func TestRotatePostureValidation(t *testing.T) {
	r := execRunner{adb: "adb"}
	// Pure validation runs before any adb call: no device needed.
	if err := r.Rotate("s", 45); err == nil {
		t.Error("bad degrees should fail fast")
	}
	if err := r.SetPosture("s", "inside-out"); err == nil {
		t.Error("bad posture should fail fast")
	}
}

func TestVideoOrdering(t *testing.T) {
	// stopCapture with no active capture fails instead of pulling garbage.
	r := execRunner{adb: "adb"}
	if _, err := stopCapture(r, "nosuch", "video", "/sdcard/x.mp4"); err == nil {
		t.Error("stop without start should fail")
	}
	// Double start is rejected (would orphan the first capture).
	captures["s/video"] = &capture{kind: "video"}
	defer delete(captures, "s/video")
	if err := r.StartVideo("s"); err == nil {
		t.Error("double start should fail")
	}
}

func TestRefreshReapsDeleted(t *testing.T) {
	r := &fakeRunner{serial: "emulator-5554", tree: []byte(`{}`), shot: []byte("png")}
	p := newMemPoster()
	p.notFoundIDs["emu_gone"] = true
	h := &host{fc: p, pool: "linux-1", runner: r, serials: map[string]string{"emu_gone": "emulator-5554"}}
	h.refresh("emu_gone", "emulator-5554")
	if _, ok := h.serials["emu_gone"]; ok {
		t.Error("deleted emu still tracked")
	}
	if len(r.kills) != 1 || r.kills[0] != "emulator-5554" {
		t.Errorf("local instance not destroyed: %v", r.kills)
	}
}
