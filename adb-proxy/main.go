// SPDX-License-Identifier: FSL-1.1-ALv2
// Copyright 2026 ImBIOS — licensed under FSL-1.1-ALv2, see LICENSE.md.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// adb-proxy runs on the AGENT side (cloud sandbox or laptop): it forwards a
// local port to the cloud emulator's adb server, so plain `adb`, Appium,
// Maestro and scrcpy work unchanged. Auth uses the org API key
// (SIDEKICK_TOKEN), like every other agent verb.

// emuTarget resolves the emu's adb_target via the control-plane.
func emuTarget(control, token, emuID string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, strings.TrimSuffix(control, "/")+"/v1/emus/"+emuID, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("emu %s: HTTP %d", emuID, resp.StatusCode)
	}
	var emu struct {
		Status    string `json:"status"`
		ADBTarget string `json:"adb_target"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&emu); err != nil {
		return "", err
	}
	if emu.ADBTarget == "" {
		return "", fmt.Errorf("emu %s has no adb target yet (status %s)", emuID, emu.Status)
	}
	return emu.ADBTarget, nil
}

// forward bridges one inbound connection to target, both directions.
func forward(in net.Conn, target string) {
	out, err := net.DialTimeout("tcp", target, 10*time.Second)
	if err != nil {
		in.Close()
		return
	}
	go func() {
		_, _ = io.Copy(out, in)
		out.Close()
	}()
	_, _ = io.Copy(in, out)
	in.Close()
}

// serve listens on local and forwards every connection to target.
// It returns only on listener failure; per-connection errors just drop.
func serve(ln net.Listener, target string) error {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		go forward(conn, target)
	}
}

func main() {
	control := flag.String("control", os.Getenv("SIDEKICK_CONTROL"), "control-plane URL")
	token := flag.String("token", os.Getenv("SIDEKICK_TOKEN"), "org API key")
	emu := flag.String("emu", "", "emulator id")
	local := flag.String("local", "127.0.0.1:5555", "local listen address")
	flag.Parse()

	if *control == "" || *token == "" || *emu == "" {
		fmt.Fprintln(os.Stderr, "adb-proxy: need --control, --token (SIDEKICK_TOKEN) and --emu")
		os.Exit(1)
	}
	target, err := emuTarget(*control, *token, *emu)
	if err != nil {
		fmt.Fprintln(os.Stderr, "adb-proxy:", err)
		fmt.Fprintln(os.Stderr, "hint: the emu may still be provisioning; retry until running")
		os.Exit(1)
	}
	ln, err := net.Listen("tcp", *local)
	if err != nil {
		fmt.Fprintln(os.Stderr, "adb-proxy: listen:", err)
		os.Exit(1)
	}
	fmt.Printf("adb-proxy: %s -> %s (adb devices, Appium, Maestro just work)\n", *local, target)
	if err := serve(ln, target); err != nil {
		fmt.Fprintln(os.Stderr, "adb-proxy:", err)
		os.Exit(1)
	}
}
