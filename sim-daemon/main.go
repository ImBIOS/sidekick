package main

import (
	"fmt"
	"os/exec"
)

// Runs on each Mac host. Wraps simctl + ScreenCaptureKit -> LiveKit + idb AX tree.
func main() { fmt.Println("sim-daemon: waiting for claims...") }

func createSim(name, device, runtime string) (string, error) {
	out, err := exec.Command("xcrun", "simctl", "create", name, device, runtime).CombinedOutput()
	return string(out), err
}
