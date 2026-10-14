// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 ImBIOS — licensed under Apache-2.0, see LICENSE-APACHE.md.
package main

import (
	"embed"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/ImBIOS/sidekick/internal/api"
	"github.com/ImBIOS/sidekick/mcp"
)

//go:embed skill-embed/SKILL.md
var skillFS embed.FS

// mcpCmd runs the MCP server on stdio: `sidekick mcp` is what
// `claude mcp add sidekick -- sidekick mcp` spawns per session.
func mcpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "run the MCP server on stdio (Claude Code, Cursor, Codex)",
		RunE: func(cmd *cobra.Command, args []string) error {
			url, token := resolveConn(cmd)
			if token == "" {
				return fmt.Errorf("not logged in: run `sidekick login` or set SIDEKICK_TOKEN")
			}
			return mcp.New(api.New(url, token)).Run(os.Stdin, os.Stdout)
		},
	}
}

// agentSetupCmd installs the skill file where coding agents look and prints
// the one-line MCP registration for each supported client.
func agentSetupCmd() *cobra.Command {
	var dir string
	c := &cobra.Command{
		Use:   "agent-setup",
		Short: "install the skill + print MCP config for your agent",
		RunE: func(cmd *cobra.Command, args []string) error {
			raw, err := skillFS.ReadFile("skill-embed/SKILL.md")
			if err != nil {
				return err
			}
			targets := []string{}
			if home, err := os.UserHomeDir(); err == nil {
				targets = append(targets, filepath.Join(home, ".claude", "skills", "sidekick", "SKILL.md"))
			}
			if dir != "" {
				targets = append(targets,
					filepath.Join(dir, ".claude", "skills", "sidekick", "SKILL.md"),
					filepath.Join(dir, "AGENTS.md"),
				)
			}
			installed := []string{}
			for _, t := range targets {
				if strings.HasSuffix(t, "AGENTS.md") {
					if err := appendNudge(t); err != nil {
						return err
					}
					installed = append(installed, t+" (nudge appended)")
					continue
				}
				if err := os.MkdirAll(filepath.Dir(t), 0o755); err != nil {
					return err
				}
				if err := os.WriteFile(t, raw, 0o644); err != nil {
					return err
				}
				installed = append(installed, t)
			}
			for _, t := range installed {
				fmt.Println("installed", t)
			}
			fmt.Println()
			fmt.Println("Claude Code: claude mcp add sidekick -- sidekick mcp")
			fmt.Println(`Cursor/Codex MCP JSON: {"mcpServers":{"sidekick":{"command":"sidekick","args":["mcp"],"env":{"SIDEKICK_TOKEN":"sk_live_..."}}}}`)
			return nil
		},
	}
	c.Flags().StringVar(&dir, "dir", "", "also install into a project dir (.claude/skills + AGENTS.md nudge)")
	return c
}
const agentsNudge = "\n## Sidekick\nThis repo builds a mobile app. Your sandbox has no Xcode or emulator: use the `sidekick` skill for all builds, simulators, and device verification (preview link + demo video on every PR).\n"

// appendNudge adds the Sidekick paragraph to AGENTS.md once (idempotent).
func appendNudge(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if strings.Contains(string(raw), "## Sidekick") {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(agentsNudge)
	return err
}

// devCmd boots one live device and prints its share URL (and JSON with
// --json): the fastest path from zero to "my agent holds the phone".
func devCmd() *cobra.Command {
	var device, runtime string
	var asJSON bool
	var timeout time.Duration
	c := &cobra.Command{
		Use:   "dev",
		Short: "boot a live device and print its share URL",
		RunE: func(cmd *cobra.Command, args []string) error {
			url, token := resolveConn(cmd)
			conn := api.New(url, token)
			var sim struct {
				ID        string `json:"id"`
				Status    string `json:"status"`
				WebRTCURL string `json:"webrtc_url"`
				AXTreeURL string `json:"ax_tree_url"`
			}
			if err := conn.Call(http.MethodPost, "/v1/sims",
				map[string]string{"device": firstStr(device, "iPhone 16"), "runtime": runtime}, &sim); err != nil {
				return err
			}
			deadline := time.Now().Add(timeout)
			for sim.Status != "running" {
				if time.Now().After(deadline) {
					return fmt.Errorf("sim %s still %s after %s; check `sidekick ios get %s`", sim.ID, sim.Status, timeout, sim.ID)
				}
				time.Sleep(2 * time.Second)
				if err := conn.Call(http.MethodGet, "/v1/sims/"+sim.ID, nil, &sim); err != nil {
					return err
				}
			}
			if asJSON {
				printJSON(sim)
				return nil
			}
			fmt.Printf("live device %s\nshare: %s\nax-tree: sidekick ios ax-tree %s\n", sim.ID, sim.WebRTCURL, sim.ID)
			return nil
		},
	}
	c.Flags().StringVar(&device, "device", "", "device name (default iPhone 16)")
	c.Flags().StringVar(&runtime, "runtime", "", "iOS runtime (default latest)")
	c.Flags().BoolVar(&asJSON, "json", false, "print the sim object as JSON")
	c.Flags().DurationVar(&timeout, "timeout", 5*time.Minute, "max wait for running")
	return c
}

func firstStr(s, d string) string {
	if s != "" {
		return s
	}
	return d
}
