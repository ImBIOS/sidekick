// SPDX-License-Identifier: FSL-1.1-ALv2
// Copyright 2026 ImBIOS — licensed under FSL-1.1-ALv2, see LICENSE.md.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ImBIOS/sidekick/internal/fleet"
)

// Device controls one booted simulator. execDevice shells simctl/idb;
// tests substitute fakeDevice. The interface keeps every verb testable
// without a Mac while the real commands stay one implementation away.
type Device interface {
	// Create boots a simulator for device/runtime, returning its UDID.
	Create(device, runtime string) (string, error)
	// Tree returns the current AX tree JSON.
	Tree(udid string) ([]byte, error)
	// Apply executes one queued verb body, returning a human result.
	// Recognized keys: tap, type, swipe, shot, record.
	Apply(udid string, body map[string]string) (string, error)
	// Shot captures a PNG screenshot.
	Shot(udid string) ([]byte, error)
	// StartRecord begins an MP4 capture; StopRecord ends it and returns bytes.
	StartRecord(udid string) error
	StopRecord(udid string) ([]byte, error)
	// Destroy tears the simulator down.
	Destroy(udid string) error
}

// poster is the slice of the fleet client the sim loop needs.
type poster interface {
	SimReady(id, host, webrtcURL string) error
	PutSimTree(id string, tree []byte) error
	PutSimShot(id string, png []byte) error
	PostSimLogs(id string, lines []string) error
	PostSimVideo(id string, mp4 []byte) error
	PostActionResult(actionID, result string) error
}

// execDevice drives real simulators. Taps/types/swipes/screenshots go
// through idb (companion `pip install idb`); lifecycle through simctl.
type execDevice struct{}

func idb(udid string, args ...string) ([]byte, error) {
	full := append([]string{"ui", "--udid", udid}, args...)
	return exec.Command("idb", full...).CombinedOutput()
}

func (execDevice) Create(device, runtime string) (string, error) {
	args := []string{"create", "sidekick", device}
	if runtime != "" {
		args = append(args, runtime)
	}
	out, err := exec.Command("xcrun", append([]string{"simctl"}, args...)...).CombinedOutput()
	udid := strings.TrimSpace(string(out))
	if err != nil {
		return "", fmt.Errorf("simctl create: %w (%s)", err, udid)
	}
	if out, err := exec.Command("xcrun", "simctl", "boot", udid).CombinedOutput(); err != nil {
		return "", fmt.Errorf("simctl boot: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return udid, nil
}

func (execDevice) Tree(udid string) ([]byte, error) {
	return idb(udid, "describe", "--json")
}

func (d execDevice) Apply(udid string, body map[string]string) (string, error) {
	switch {
	case body["tap"] != "":
		out, err := idb(udid, "tap", body["tap"])
		return fmt.Sprintf("tapped %s (%s)", body["tap"], strings.TrimSpace(string(out))), err
	case body["type"] != "":
		out, err := idb(udid, "text", body["type"])
		return fmt.Sprintf("typed %d chars", len(body["type"])), wrapErr(err, out)
	case body["swipe"] != "":
		out, err := idb(udid, "swipe", body["swipe"])
		return fmt.Sprintf("swiped %s", body["swipe"]), wrapErr(err, out)
	default:
		return "", fmt.Errorf("unknown verb: %v", body)
	}
}

func wrapErr(err error, out []byte) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w (%s)", err, strings.TrimSpace(string(out)))
}

func (execDevice) Shot(udid string) ([]byte, error) {
	path := os.TempDir() + "/sidekick-" + udid + ".png"
	out, err := exec.Command("xcrun", "simctl", "io", udid, "screenshot", path).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("screenshot: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	defer os.Remove(path)
	return os.ReadFile(path)
}

// recordProcs tracks background `simctl io recordVideo` captures per UDID.
var recordProcs = map[string]*exec.Cmd{}

func (execDevice) StartRecord(udid string) error {
	path := os.TempDir() + "/sidekick-" + udid + ".mp4"
	cmd := exec.Command("xcrun", "simctl", "io", udid, "recordVideo", "--codec=h264", path)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("recordVideo: %w", err)
	}
	recordProcs[udid] = cmd
	return nil
}

func (execDevice) StopRecord(udid string) ([]byte, error) {
	cmd, ok := recordProcs[udid]
	if !ok {
		return nil, fmt.Errorf("no active recording for %s", udid)
	}
	delete(recordProcs, udid)
	_ = cmd.Process.Signal(os.Interrupt)
	_ = cmd.Wait()
	path := os.TempDir() + "/sidekick-" + udid + ".mp4"
	defer os.Remove(path)
	return os.ReadFile(path)
}

func (execDevice) Destroy(udid string) error {
	out, err := exec.Command("xcrun", "simctl", "shutdown", udid).CombinedOutput()
	if err != nil && !strings.Contains(string(out), "Invalid device state") {
		return fmt.Errorf("shutdown: %w", err)
	}
	out, err = exec.Command("xcrun", "simctl", "delete", udid).CombinedOutput()
	if err != nil {
		return fmt.Errorf("delete: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// provision boots the claimed sim and publishes its first tree + screenshot.
func provision(fc poster, host, streamBase, simID, device, runtime string, dev Device) (string, error) {
	udid, err := dev.Create(device, runtime)
	if err != nil {
		return "", err
	}
	if err := fc.SimReady(simID, host, streamBase+"/"+simID); err != nil {
		return "", err
	}
	if tree, err := dev.Tree(udid); err == nil {
		_ = fc.PutSimTree(simID, tree)
	} else {
		_ = fc.PostSimLogs(simID, []string{"[sidekick] initial tree failed: " + err.Error()})
	}
	if png, err := dev.Shot(udid); err == nil {
		_ = fc.PutSimShot(simID, png)
	}
	return udid, nil
}

// applyOne executes a queued action and publishes result + fresh tree.
// Screenshot refreshes after every verb so agents always see the outcome.
func applyOne(fc poster, simID, actionID, udid string, body map[string]string, dev Device) {
	var result string
	if op, ok := body["record"]; ok {
		var err error
		if op == "start" {
			err = dev.StartRecord(udid)
			result = "recording started"
		} else {
			var mp4 []byte
			mp4, err = dev.StopRecord(udid)
			if err == nil {
				_ = fc.PostSimVideo(simID, mp4)
				result = fmt.Sprintf("recording stopped (%d bytes)", len(mp4))
			}
		}
		if err != nil {
			result = "record failed: " + err.Error()
		}
	} else {
		var err error
		result, err = dev.Apply(udid, body)
		if err != nil {
			result = "apply failed: " + err.Error()
		}
	}
	_ = fc.PostActionResult(actionID, result)
	if tree, err := dev.Tree(udid); err == nil {
		_ = fc.PutSimTree(simID, tree)
	}
	if png, err := dev.Shot(udid); err == nil {
		_ = fc.PutSimShot(simID, png)
	} else {
		_ = fc.PostSimLogs(simID, []string{"[sidekick] screenshot failed: " + err.Error()})
	}
}

func main() {
	control := flag.String("control", "", "control-plane URL (or SIDEKICK_CONTROL)")
	token := flag.String("token", "", "fleet host token (or SIDEKICK_HOST_TOKEN)")
	pool := flag.String("pool", "", "pool name (or SIDEKICK_POOL)")
	regionflag := flag.String("region", "", "region pin, e.g. asia-southeast1 (or SIDEKICK_REGION)")
	stream := flag.String("stream", "", "WebRTC base URL agents open (default <control>/webrtc)")
	interval := flag.Duration("interval", time.Second, "claim poll interval")
	flag.Parse()

	ctrl, tok, pl, region, err := fleet.Resolve(*control, *token, *pool, *regionflag)
	if err != nil {
		fmt.Fprintln(os.Stderr, "sim-daemon:", err)
		os.Exit(1)
	}
	streamBase := *stream
	if streamBase == "" {
		streamBase = ctrl + "/webrtc"
	}
	fc := fleet.New(ctrl, tok)
	dev := execDevice{}
	// udids tracks provisioned sims across loop iterations. Real fleets
	// persist this (reboot recovery); MVP keeps it in memory.
	udids := map[string]string{}
	fmt.Printf("sim-daemon: pool %s via %s\n", pl, ctrl)
	for {
		claim, ok, err := fc.ClaimNext(pl, region, []string{"sim", "sim-action"})
		if err != nil {
			fmt.Fprintln(os.Stderr, "sim-daemon: claim:", err)
			time.Sleep(*interval)
			continue
		}
		if !ok {
			time.Sleep(*interval)
			continue
		}
		switch claim.Kind {
		case "sim":
			simID := claim.SimID()
			device, _ := claim.Sim["device"].(string)
			runtime, _ := claim.Sim["runtime"].(string)
			udid, err := provision(fc, pl, streamBase, simID, device, runtime, dev)
			if err != nil {
				fmt.Fprintln(os.Stderr, "sim-daemon: provision:", err)
				_ = fc.PostSimLogs(simID, []string{"[sidekick] provision failed: " + err.Error()})
				continue
			}
			udids[simID] = udid
		case "sim-action":
			simID := claim.Action.SimID
			udid, ok := udids[simID]
			if !ok {
				_ = fc.PostActionResult(claim.Action.ID, "host restarted; sim UDID lost, recreate the sim")
				continue
			}
			applyOne(fc, simID, claim.Action.ID, udid, claim.Action.Body, dev)
		}
	}
}
