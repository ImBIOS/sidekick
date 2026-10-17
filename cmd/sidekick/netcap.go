// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 ImBIOS — licensed under Apache-2.0, see LICENSE-APACHE.md.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// netcapCmd captures device network traffic to a pcap file: it runs tcpdump
// on the device (needs adb root — emulator images allow `adb root`) for the
// given duration, then pulls the file. Foreground by design: CI wraps it.
func netcapCmd() *cobra.Command {
	var out, serial, adbBin string
	var duration time.Duration
	c := &cobra.Command{
		Use:   "netcap --out cap.pcap",
		Short: "capture device traffic to pcap (needs adb root)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if out == "" {
				return fmt.Errorf("give --out cap.pcap")
			}
			if serial == "" {
				serial = "localhost:5555"
			}
			run := func(args ...string) ([]byte, error) {
				full := append([]string{"-s", serial}, args...)
				return exec.Command(adbBin, full...).CombinedOutput()
			}
			if outb, err := run("root"); err != nil {
				return fmt.Errorf("adb root (needed for capture): %w (%s)", err, firstLine(outb))
			}
			devicePath := "/sdcard/sidekick-net.pcap"
			// Start tcpdump in background; poll until it appears.
			bg := exec.Command(adbBin, "-s", serial, "shell", "tcpdump", "-w", devicePath, "-s", "0")
			if err := bg.Start(); err != nil {
				return fmt.Errorf("tcpdump start: %w (is tcpdump on this image?)", err)
			}
			deadline := time.Now().Add(10 * time.Second)
			for {
				if _, err := run("shell", "ls", devicePath); err == nil {
					break
				}
				if time.Now().After(deadline) {
					_ = bg.Process.Kill()
					return fmt.Errorf("tcpdump never produced %s (missing binary?)", devicePath)
				}
				time.Sleep(time.Second)
			}
			fmt.Printf("capturing %s on %s...\n", duration, serial)
			time.Sleep(duration)
			_ = bg.Process.Kill()
			_ = bg.Wait()
			raw, err := run("exec-out", "cat", devicePath)
			if err != nil || len(raw) < 24 {
				return fmt.Errorf("pull capture: %w (%d bytes)", err, len(raw))
			}
			if _, err := run("shell", "rm", "-f", devicePath); err != nil {
				fmt.Fprintln(os.Stderr, "warning: device cleanup:", err)
			}
			if err := os.WriteFile(out, raw, 0o644); err != nil {
				return err
			}
			fmt.Printf("saved %d bytes to %s\n", len(raw), out)
			return nil
		},
	}
	c.Flags().StringVar(&out, "out", "", "pcap file (required)")
	c.Flags().StringVar(&serial, "adb-serial", "", "adb target (default localhost:5555, the adb-proxy tunnel)")
	c.Flags().StringVar(&adbBin, "adb", "adb", "adb binary")
	c.Flags().DurationVar(&duration, "duration", 30*time.Second, "capture length")
	return c
}

func firstLine(b []byte) string {
	s := string(b)
	for i, ch := range s {
		if ch == '\n' {
			return s[:i]
		}
	}
	return s
}

// recordCmd captures an MP4 over the adb tunnel (foreground: records for
// --duration, then pulls). Screenrecord caps at 180s per invocation.
func droidRecordCmd() *cobra.Command {
	var out, serial, adbBin string
	var duration time.Duration
	c := &cobra.Command{
		Use:   "record <id> --out demo.mp4",
		Short: "record the emulator screen to mp4",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if out == "" {
				return fmt.Errorf("give --out demo.mp4")
			}
			if duration <= 0 || duration > 180*time.Second {
				return fmt.Errorf("duration must be 1s..180s (platform cap)")
			}
			if serial == "" {
				serial = "localhost:5555"
			}
			// Fail fast when the tunnel/device is unreachable instead of
			// recording silence into a missing file.
			if out, err := exec.Command(adbBin, "-s", serial, "get-state").CombinedOutput(); err != nil || strings.TrimSpace(string(out)) != "device" {
				return fmt.Errorf("adb %s not ready (%s): is adb-proxy running?", serial, firstLine(out))
			}
			devicePath := "/sdcard/sidekick-rec.mp4"
			run := func(a ...string) ([]byte, error) {
				full := append([]string{"-s", serial}, a...)
				return exec.Command(adbBin, full...).CombinedOutput()
			}
			bg := exec.Command(adbBin, "-s", serial, "shell", "screenrecord", "--time-limit", "180", devicePath)
			if err := bg.Start(); err != nil {
				return fmt.Errorf("screenrecord start: %w", err)
			}
			time.Sleep(duration)
			_ = bg.Process.Kill()
			_ = bg.Wait()
			raw, err := run("exec-out", "cat", devicePath)
			if err != nil || len(raw) < 1024 {
				return fmt.Errorf("pull recording: %w (%d bytes)", err, len(raw))
			}
			_, _ = run("shell", "rm", "-f", devicePath)
			if err := os.WriteFile(out, raw, 0o644); err != nil {
				return err
			}
			fmt.Printf("saved %d bytes to %s\n", len(raw), out)
			return nil
		},
	}
	c.Flags().StringVar(&out, "out", "", "mp4 file (required)")
	c.Flags().StringVar(&serial, "adb-serial", "", "adb target (default localhost:5555)")
	c.Flags().StringVar(&adbBin, "adb", "adb", "adb binary")
	c.Flags().DurationVar(&duration, "duration", 30*time.Second, "record length (max 180s)")
	return c
}

// rotateCmd sets display rotation over the adb tunnel (no Mac, no host
// round-trip: the agent drives it directly like any adb command).
func rotateCmd() *cobra.Command {
	var serial, adbBin string
	c := &cobra.Command{
		Use:   "rotate <id> <0|90|180|270>",
		Short: "rotate the emulator display",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			idx := map[string]string{"0": "0", "90": "1", "180": "2", "270": "3"}[args[1]]
			if idx == "" {
				return fmt.Errorf("degrees must be one of 0, 90, 180, 270")
			}
			if serial == "" {
				serial = "localhost:5555"
			}
			run := func(a ...string) error {
				full := append([]string{"-s", serial}, a...)
				if out, err := exec.Command(adbBin, full...).CombinedOutput(); err != nil {
					return fmt.Errorf("adb %s: %w (%s)", strings.Join(a, " "), err, firstLine(out))
				}
				return nil
			}
			if err := run("shell", "settings", "put", "system", "accelerometer_rotation", "0"); err != nil {
				return err
			}
			if err := run("shell", "settings", "put", "system", "user_rotation", idx); err != nil {
				return err
			}
			fmt.Printf("rotated %s to %s\n", args[0], args[1])
			return nil
		},
	}
	c.Flags().StringVar(&serial, "adb-serial", "", "adb target (default localhost:5555)")
	c.Flags().StringVar(&adbBin, "adb", "adb", "adb binary")
	return c
}

// parseFoldState extracts a device-state identifier by name (twin of
// emu-host's discovery: identifiers differ per foldable, never hardcode).
func parseFoldState(out, name string) string {
	for _, line := range strings.Split(out, "\n") {
		id, found := parseStateLine(line)
		if found == "" {
			continue
		}
		if found == name || normState(found) == normState(name) {
			return id
		}
	}
	return ""
}

// parseStateLine pulls (identifier, name) from one print-states line.
func parseStateLine(line string) (string, string) {
	i := strings.Index(line, "identifier=")
	if i < 0 {
		return "", ""
	}
	rest := line[i+len("identifier="):]
	j := 0
	for j < len(rest) && rest[j] >= '0' && rest[j] <= '9' {
		j++
	}
	if j == 0 {
		return "", ""
	}
	id := rest[:j]
	name := ""
	if k := strings.Index(line, "name='"); k >= 0 {
		rest2 := line[k+len("name='"):]
		if q := strings.Index(rest2, "'"); q >= 0 {
			name = rest2[:q]
		}
	}
	return id, name
}

// normState treats HALF_OPEN and HALF_OPENED as the same posture.
func normState(s string) string {
	return strings.TrimSuffix(strings.TrimSuffix(s, "ED"), "_")
}
// postureCmd requests a foldable posture via the platform device_state
// controller. Fails loudly on non-foldables instead of pretending.
func postureCmd() *cobra.Command {
	var serial, adbBin string
	c := &cobra.Command{
		Use:   "posture <id> <open|half-open|closed>",
		Short: "set foldable hinge posture",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			want := map[string]string{"closed": "CLOSED", "half-open": "HALF_OPENED", "open": "OPENED"}[args[1]]
			if want == "" {
				return fmt.Errorf(`posture must be one of "open", "half-open", "closed"`)
			}
			if serial == "" {
				serial = "localhost:5555"
			}
			run := func(a ...string) ([]byte, error) {
				full := append([]string{"-s", serial}, a...)
				return exec.Command(adbBin, full...).CombinedOutput()
			}
			// Identifiers differ per device: discover, don't hardcode.
			states, err := run("shell", "cmd", "device_state", "print-states")
			if err != nil {
				return fmt.Errorf("posture unsupported on this device (%s)", firstLine(states))
			}
			id := parseFoldState(string(states), want)
			if id == "" {
				return fmt.Errorf("posture %q unsupported on this device", args[1])
			}
			if out, err := run("shell", "cmd", "device_state", "state", id); err != nil {
				return fmt.Errorf("posture %q failed (%s)", args[1], firstLine(out))
			}
			fmt.Printf("posture %s -> %s\n", args[0], args[1])
			return nil
		},
	}
	c.Flags().StringVar(&serial, "adb-serial", "", "adb target (default localhost:5555)")
	c.Flags().StringVar(&adbBin, "adb", "adb", "adb binary")
	return c
}
