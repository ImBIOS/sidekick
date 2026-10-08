// SPDX-License-Identifier: FSL-1.1-ALv2
// Copyright 2026 ImBIOS — licensed under FSL-1.1-ALv2, see LICENSE.md.
package main

import "fmt"

// adb-proxy: dials control-plane, forwards remote emulator to local :5555
// so `adb devices`, Appium, Maestro, scrcpy just work.
func main() { fmt.Println("adb-proxy: forward tcp:5037") }
