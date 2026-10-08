// SPDX-License-Identifier: FSL-1.1-ALv2
// Copyright 2026 ImBIOS — licensed under FSL-1.1-ALv2, see LICENSE.md.
package main

import (
	"fmt"
	"os/exec"
)

// Runs on each Mac host. Pulls source, reuses warm DerivedData, streams xcodebuild.
func main() {
	fmt.Println("build-daemon: waiting for claims...")
}

// Build runs xcodebuild with warm cache. Called per claim.
func Build(workdir, scheme, derivedData string) *exec.Cmd {
	return exec.Command("xcodebuild", "-scheme", scheme,
		"-destination", "generic/platform=iOS Simulator",
		"-derivedDataPath", derivedData, "build")
}
