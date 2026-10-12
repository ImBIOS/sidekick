// SPDX-License-Identifier: FSL-1.1-ALv2
// Copyright 2026 ImBIOS — licensed under FSL-1.1-ALv2, see LICENSE.md.
package main

import (
	"fmt"
	"strings"
	"testing"
)

// fakeDevice pretends to be simctl/idb with canned state.
type fakeDevice struct {
	udid  string
	tree  string
	shot  []byte
	video []byte
	rec   bool
	log   []string
}

func (f *fakeDevice) Create(device, runtime string) (string, error) {
	if device == "" {
		return "", fmt.Errorf("device is required")
	}
	f.udid = "UDID-" + device
	f.log = append(f.log, "create "+device)
	return f.udid, nil
}

func (f *fakeDevice) Tree(udid string) ([]byte, error) {
	return []byte(f.tree), nil
}

func (f *fakeDevice) Apply(udid string, body map[string]string) (string, error) {
	if body["tap"] != "" {
		f.tree = strings.Replace(f.tree, "@e1", "@e9", 1)
		return "tapped " + body["tap"], nil
	}
	if body["type"] != "" {
		return fmt.Sprintf("typed %d chars", len(body["type"])), nil
	}
	return "", fmt.Errorf("unknown verb: %v", body)
}

func (f *fakeDevice) Shot(udid string) ([]byte, error) { return f.shot, nil }

func (f *fakeDevice) StartRecord(udid string) error {
	f.rec = true
	return nil
}

func (f *fakeDevice) StopRecord(udid string) ([]byte, error) {
	if !f.rec {
		return nil, fmt.Errorf("not recording")
	}
	f.rec = false
	return f.video, nil
}

func (f *fakeDevice) Destroy(udid string) error { return nil }

// memPoster records control-plane posts.
type memPoster struct {
	ready  map[string]string
	trees  map[string]string
	shots  map[string]int
	logs   map[string][]string
	videos map[string]int
	acts   map[string]string
}

func newMemPoster() *memPoster {
	return &memPoster{
		ready: map[string]string{}, trees: map[string]string{},
		shots: map[string]int{}, logs: map[string][]string{},
		videos: map[string]int{}, acts: map[string]string{},
	}
}

func (m *memPoster) SimReady(id, host, url string) error {
	m.ready[id] = host + "|" + url
	return nil
}
func (m *memPoster) PutSimTree(id string, tree []byte) error {
	m.trees[id] = string(tree)
	return nil
}
func (m *memPoster) PutSimShot(id string, png []byte) error {
	m.shots[id] = len(png)
	return nil
}
func (m *memPoster) PostSimLogs(id string, lines []string) error {
	m.logs[id] = append(m.logs[id], lines...)
	return nil
}
func (m *memPoster) PostSimVideo(id string, mp4 []byte) error {
	m.videos[id] = len(mp4)
	return nil
}
func (m *memPoster) PostActionResult(aid, result string) error {
	m.acts[aid] = result
	return nil
}

func TestProvision(t *testing.T) {
	dev := &fakeDevice{tree: `{"root":"@e1"}`, shot: []byte("png")}
	p := newMemPoster()
	udid, err := provision(p, "mac-1", "wss://x", "sim_1", "iPhone 16", "", dev)
	if err != nil || udid != "UDID-iPhone 16" {
		t.Fatalf("provision: %v %q", err, udid)
	}
	if p.ready["sim_1"] != "mac-1|wss://x/sim_1" {
		t.Errorf("ready: %q", p.ready["sim_1"])
	}
	if p.trees["sim_1"] != `{"root":"@e1"}` || p.shots["sim_1"] != 3 {
		t.Errorf("initial state: tree=%q shot=%d", p.trees["sim_1"], p.shots["sim_1"])
	}
}

func TestApplyOne(t *testing.T) {
	dev := &fakeDevice{tree: `{"root":"@e1"}`, shot: []byte("png")}
	p := newMemPoster()
	applyOne(p, "sim_1", "act_1", "UDID-x", map[string]string{"tap": "@e1"}, dev)
	if p.acts["act_1"] != "tapped @e1" {
		t.Errorf("result: %q", p.acts["act_1"])
	}
	// Tree + screenshot refresh after every verb.
	if !strings.Contains(p.trees["sim_1"], "@e9") {
		t.Errorf("tree not refreshed: %q", p.trees["sim_1"])
	}
	if p.shots["sim_1"] != 3 {
		t.Errorf("shot not refreshed: %d", p.shots["sim_1"])
	}
	// Unknown verbs report failure, never crash the loop.
	applyOne(p, "sim_1", "act_2", "UDID-x", map[string]string{"dance": "now"}, dev)
	if !strings.Contains(p.acts["act_2"], "apply failed") {
		t.Errorf("bad verb result: %q", p.acts["act_2"])
	}
}

func TestApplyRecord(t *testing.T) {
	dev := &fakeDevice{tree: `{}`, shot: []byte("png"), video: []byte("mp4bytes")}
	p := newMemPoster()
	applyOne(p, "sim_1", "a1", "UDID-x", map[string]string{"record": "start"}, dev)
	if p.acts["a1"] != "recording started" {
		t.Errorf("start: %q", p.acts["a1"])
	}
	applyOne(p, "sim_1", "a2", "UDID-x", map[string]string{"record": "stop"}, dev)
	if p.videos["sim_1"] != 8 {
		t.Errorf("video bytes: %d", p.videos["sim_1"])
	}
}
