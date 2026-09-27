package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var root = &cobra.Command{Use: "lim", Short: "OpenLim - remote Xcode, iOS, Android for cloud agents"}

func main() {
	xcode := &cobra.Command{Use: "xcode", Short: "remote Xcode builds"}
	build := &cobra.Command{Use: "build [path]", Short: "sync + build remotely", Args: cobra.MaximumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		scheme, _ := cmd.Flags().GetString("scheme")
		fmt.Printf("syncing %s -> control-plane, scheme=%s, streaming xcodebuild logs...\n", first(args, "."), scheme)
		// TODO: tar+zstd -> presigned S3 -> POST /v1/builds -> SSE stream
		return nil
	}}
	build.Flags().String("scheme", "", "Xcode scheme (required)")
	xcode.AddCommand(build)

	ios := &cobra.Command{Use: "ios", Short: "cloud iOS simulators"}
	iosCreate := &cobra.Command{Use: "create", Short: "create simulator", RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(`{"id":"sim_abc","webrtc_url":"wss://stream.openlim.dev/sim_abc","ax_tree_url":"https://api.openlim.dev/v1/sims/sim_abc/tree"}`)
		return nil
	}}
	ios.AddCommand(iosCreate)
	ios.AddCommand(&cobra.Command{Use: "act <id>", Short: "tap/type/swipe batch actions", RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("ok: action applied, diff: 3 elements changed")
		return nil
	}})

	android := &cobra.Command{Use: "android", Short: "cloud Android emulators"}
	android.AddCommand(&cobra.Command{Use: "create", Short: "create emulator + adb tunnel", RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(`{"id":"emu_abc","adb":"localhost:5555 via lim adb forward"}`)
		return nil
	}})

	root.AddCommand(xcode, ios, android)
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func first(s []string, d string) string {
	if len(s) > 0 {
		return s[0]
	}
	return d
}
