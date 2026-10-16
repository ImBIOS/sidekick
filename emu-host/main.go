// SPDX-License-Identifier: FSL-1.1-ALv2
// Copyright 2026 ImBIOS — licensed under FSL-1.1-ALv2, see LICENSE.md.
package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/ImBIOS/sidekick/internal/fleet"
)

// Runner boots emulators and reads them back. execRunner shells the
// Android SDK (emulator + adb); tests substitute fakeRunner. Agent verbs
// (tap/type/…) travel over the adb tunnel via adb-proxy — the host only
// provisions, reports tree/screenshots/logs, and tears down.
type Runner interface {
	// Boot starts profile (an AVD name) and returns its adb serial once
	// sys.boot_completed=1 (or an error after timeout).
	Boot(profile string) (serial string, err error)
	// Tree dumps the UI hierarchy as agent JSON (@e refs).
	Tree(serial string) ([]byte, error)
	// Shot captures a PNG screenshot.
	Shot(serial string) ([]byte, error)
	// Logs returns recent logcat lines.
	Logs(serial string) ([]string, error)
	// Destroy kills the emulator.
	Destroy(serial string) error
	// StartVideo begins an MP4 capture (adb screenrecord); StopVideo ends
	// it and returns the bytes.
	StartVideo(serial string) error
	StopVideo(serial string) ([]byte, error)
	// Rotate sets the display rotation (0|90|180|270).
	Rotate(serial string, degrees int) error
	// SetPosture requests a foldable posture (open|half-open|closed).
	SetPosture(serial, posture string) error
	// StartNetcap begins a pcap capture; StopNetcap ends it and returns bytes.
	StartNetcap(serial string) error
	StopNetcap(serial string) ([]byte, error)
}

// poster is the fleet surface the host loop needs (memPoster in tests).
type poster interface {
	EmuReady(id, host, adbTarget string) error
	PutEmuTree(id string, tree []byte) error
	PutEmuShot(id string, png []byte) error
	PostEmuLogs(id string, lines []string) error
}

// execRunner drives real AVDs. Requires: emulator + adb on PATH, KVM on
// Linux (or HAXM/HVF on Mac — the same binary joins any pool).
type execRunner struct {
	adb         string
	bootTimeout time.Duration
}

func (r execRunner) adbCmd(serial string, args ...string) *exec.Cmd {
	full := append([]string{"-s", serial}, args...)
	return exec.Command(r.adb, full...)
}

func (r execRunner) Boot(profile string) (string, error) {
	cmd := exec.Command("emulator", "-avd", profile,
		"-no-window", "-no-audio", "-no-snapshot", "-gpu", "swiftshader_indirect")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("emulator start: %w", err)
	}
	// The first emulator instance is emulator-5554; adb sees it there.
	serial := "emulator-5554"
	deadline := time.Now().Add(r.bootTimeout)
	for {
		out, err := exec.Command(r.adb, "-s", serial, "shell", "getprop", "sys.boot_completed").CombinedOutput()
		if err == nil && strings.TrimSpace(string(out)) == "1" {
			return serial, nil
		}
		if time.Now().After(deadline) {
			return "", fmt.Errorf("emulator %s never booted (%s)", profile, strings.TrimSpace(string(out)))
		}
		time.Sleep(5 * time.Second)
	}
}

func (r execRunner) Tree(serial string) ([]byte, error) {
	out, err := r.adbCmd(serial, "exec-out", "uiautomator", "dump", "/dev/stdout").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("uiautomator: %w (%s)", err, firstLine(out))
	}
	return uiaToJSON(out)
}

func (r execRunner) Shot(serial string) ([]byte, error) {
	out, err := r.adbCmd(serial, "exec-out", "screencap", "-p").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("screencap: %w", err)
	}
	// exec-out may prefix transport noise; the PNG must start at its magic.
	if i := bytesIndex(out, []byte{0x89, 0x50, 0x4E, 0x47}); i > 0 {
		out = out[i:]
	}
	if len(out) < 8 || out[0] != 0x89 || out[1] != 0x50 {
		return nil, fmt.Errorf("screencap: not a PNG (%d bytes)", len(out))
	}
	return out, nil
}

func (r execRunner) Logs(serial string) ([]string, error) {
	out, err := r.adbCmd(serial, "logcat", "-d", "-v", "brief", "-t", "200").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("logcat: %w", err)
	}
	var lines []string
	for _, l := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(l) != "" {
			lines = append(lines, l)
		}
	}
	return lines, nil
}

func (r execRunner) Destroy(serial string) error {
	out, err := r.adbCmd(serial, "emu", "kill").CombinedOutput()
	if err != nil {
		return fmt.Errorf("emu kill: %w (%s)", err, firstLine(out))
	}
	return nil
}

// captures tracks background on-device recordings per serial (video + pcap).
type capture struct {
	cmd  *exec.Cmd
	path string
	kind string // "video" | "pcap"
}

var captures = map[string]*capture{}

// StartVideo begins `screenrecord` on the device (time-limit 180s, the
// platform cap; agents stop earlier via record --stop).
func (r execRunner) StartVideo(serial string) error {
	return startCapture(r, serial, "video", "/sdcard/sidekick-rec.mp4",
		"shell", "screenrecord", "--time-limit", "180", "/sdcard/sidekick-rec.mp4")
}

// StopVideo ends the capture and pulls the MP4.
func (r execRunner) StopVideo(serial string) ([]byte, error) {
	return stopCapture(r, serial, "video", "/sdcard/sidekick-rec.mp4")
}

// Rotate locks rotation off and sets 0/90/180/270 (user_rotation index).
func (r execRunner) Rotate(serial string, degrees int) error {
	idx := map[int]string{0: "0", 90: "1", 180: "2", 270: "3"}[degrees]
	if idx == "" {
		return fmt.Errorf("degrees must be one of 0, 90, 180, 270")
	}
	for _, args := range [][]string{
		{"shell", "settings", "put", "system", "accelerometer_rotation", "0"},
		{"shell", "settings", "put", "system", "user_rotation", idx},
	} {
		if out, err := r.adbCmd(serial, args...).CombinedOutput(); err != nil {
			return fmt.Errorf("rotate: %w (%s)", err, firstLine(out))
		}
	}
	return nil
}

// SetPosture requests a foldable posture. It drives the platform
// device_state controller when present; otherwise it fails loudly instead
// of pretending (foldables without posture control are just phones).
func (r execRunner) SetPosture(serial, posture string) error {
	id := map[string]string{"closed": "0", "half-open": "1", "open": "2"}[posture]
	if id == "" {
		return fmt.Errorf(`posture must be one of "closed", "half-open", "open"`)
	}
	out, err := r.adbCmd(serial, "shell", "cmd", "device_state", "state", id).CombinedOutput()
	if err != nil {
		return fmt.Errorf("posture %q unsupported on this device (%s)", posture, firstLine(out))
	}
	return nil
}

// StartNetcap begins a pcap capture (needs adb root; emulator images allow
// `adb root`). Agents analyze traffic for the app under test.
func (r execRunner) StartNetcap(serial string) error {
	if out, err := r.adbCmd(serial, "root").CombinedOutput(); err != nil {
		return fmt.Errorf("adb root (needed for capture): %w (%s)", err, firstLine(out))
	}
	return startCapture(r, serial, "pcap", "/sdcard/sidekick-net.pcap",
		"shell", "tcpdump", "-w", "/sdcard/sidekick-net.pcap", "-s", "0")
}

// StopNetcap ends the capture and pulls the pcap.
func (r execRunner) StopNetcap(serial string) ([]byte, error) {
	return stopCapture(r, serial, "pcap", "/sdcard/sidekick-net.pcap")
}

func startCapture(r execRunner, serial, kind, devicePath string, args ...string) error {
	if _, active := captures[serial+"/"+kind]; active {
		return fmt.Errorf("%s capture already running", kind)
	}
	cmd := r.adbCmd(serial, args...)
	cmd.Stdout, cmd.Stderr = nil, nil
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("%s start: %w", kind, err)
	}
	captures[serial+"/"+kind] = &capture{cmd: cmd, path: devicePath, kind: kind}
	return nil
}

func stopCapture(r execRunner, serial, kind, devicePath string) ([]byte, error) {
	key := serial + "/" + kind
	c, ok := captures[key]
	if !ok {
		return nil, fmt.Errorf("no active %s capture", kind)
	}
	delete(captures, key)
	if c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
	}
	_ = c.cmd.Wait()
	out, err := r.adbCmd(serial, "exec-out", "cat", devicePath).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s pull: %w", kind, err)
	}
	_, _ = r.adbCmd(serial, "shell", "rm", "-f", devicePath).CombinedOutput()
	return out, nil
}

func firstLine(b []byte) string {
	s := strings.TrimSpace(string(b))
	if i := strings.Index(s, "\n"); i >= 0 {
		return s[:i]
	}
	return s
}

func bytesIndex(b, sub []byte) int {
	for i := 0; i+len(sub) <= len(b); i++ {
		match := true
		for j := range sub {
			if b[i+j] != sub[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// publicAddr guesses how agents reach this host's adb port (flag wins).
func publicAddr(flagAddr string) string {
	if flagAddr != "" {
		return flagAddr
	}
	// UDP dial learns the outbound IP without sending traffic.
	c, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer c.Close()
	if host, _, err := net.SplitHostPort(c.LocalAddr().String()); err == nil {
		return host
	}
	return "127.0.0.1"
}

// host tracks live emulators and refreshes their reported state.
type host struct {
	fc       poster
	pool     string
	adbAddr  string
	adbPort  string
	runner   Runner
	mu       sync.Mutex
	serials  map[string]string // emuID -> adb serial
	interval time.Duration
}

// provision boots one claimed emu and publishes its first state.
func (h *host) provision(emuID, profile string) {
	serial, err := h.runner.Boot(profile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "emu-host: boot:", err)
		return
	}
	target := h.adbAddr + ":" + h.adbPort
	if err := h.fc.EmuReady(emuID, h.pool, target); err != nil {
		fmt.Fprintln(os.Stderr, "emu-host: ready:", err)
		_ = h.runner.Destroy(serial)
		return
	}
	h.mu.Lock()
	h.serials[emuID] = serial
	h.mu.Unlock()
	h.refresh(emuID, serial)
}

// refresh reposts tree/shot/logs; a 404 from the plane means the agent
// deleted the emu, so the local instance dies too.
func (h *host) refresh(emuID, serial string) {
	dead := false
	if tree, err := h.runner.Tree(serial); err == nil {
		if err := h.fc.PutEmuTree(emuID, tree); err != nil {
			dead = isNotFound(err)
		}
	}
	if !dead {
		if png, err := h.runner.Shot(serial); err == nil {
			if err := h.fc.PutEmuShot(emuID, png); err != nil {
				dead = isNotFound(err)
			}
		}
	}
	if !dead {
		if lines, err := h.runner.Logs(serial); err == nil && len(lines) > 0 {
			_ = h.fc.PostEmuLogs(emuID, lines)
		}
	}
	if dead {
		_ = h.runner.Destroy(serial)
		h.mu.Lock()
		delete(h.serials, emuID)
		h.mu.Unlock()
		fmt.Println("emu-host: reaped deleted emu", emuID)
	}
}

// isNotFound sniffs the fleet client's HTTP 404. The poster interface
// returns raw errors; matching on status text keeps it client-agnostic.
func isNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), "404")
}

func main() {
	control := flag.String("control", "", "control-plane URL (or SIDEKICK_CONTROL)")
	token := flag.String("token", "", "fleet host token (or SIDEKICK_HOST_TOKEN)")
	pool := flag.String("pool", "linux-pool", "pool name (or SIDEKICK_POOL)")
	adbBin := flag.String("adb", "adb", "adb binary")
	adbPort := flag.String("adb-port", "5555", "adb server port agents dial")
	adbAddr := flag.String("adb-addr", os.Getenv("EMU_ADB_ADDR"), "public address agents dial (default: outbound IP)")
	bootTimeout := flag.Duration("boot-timeout", 10*time.Minute, "max wait for sys.boot_completed")
	interval := flag.Duration("interval", 20*time.Second, "claim poll + refresh interval")
	flag.Parse()

	ctrl, tok, pl, err := fleet.Resolve(*control, *token, *pool)
	if err != nil {
		fmt.Fprintln(os.Stderr, "emu-host:", err)
		os.Exit(1)
	}
	fc := fleet.New(ctrl, tok)
	h := &host{
		fc: fc, pool: pl,
		adbAddr: publicAddr(*adbAddr), adbPort: *adbPort,
		runner:   execRunner{adb: *adbBin, bootTimeout: *bootTimeout},
		serials:  map[string]string{},
		interval: *interval,
	}
	fmt.Printf("emu-host: pool %s via %s (adb %s:%s)\n", pl, ctrl, h.adbAddr, h.adbPort)
	for {
		claim, ok, err := fc.ClaimNext(pl, []string{"emu"})
		if err != nil {
			fmt.Fprintln(os.Stderr, "emu-host: claim:", err)
		} else if ok {
			// Boot blocks for minutes; background it so refreshes continue.
			go h.provision(claim.EmuID(), emuProfile(claim))
		}
		h.mu.Lock()
		snapshot := map[string]string{}
		for id, serial := range h.serials {
			snapshot[id] = serial
		}
		h.mu.Unlock()
		for id, serial := range snapshot {
			h.refresh(id, serial)
		}
		time.Sleep(h.interval)
	}
}

func emuProfile(claim *fleet.Claim) string {
	if p, ok := claim.Emu["profile"].(string); ok {
		return p
	}
	return ""
}
