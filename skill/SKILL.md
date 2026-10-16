<!-- SPDX-License-Identifier: Apache-2.0 | Copyright 2026 ImBIOS — see LICENSE-APACHE.md -->
# Sidekick agent skill — drop into ~/.claude/skills/sidekick/ or .claude/skills/
# Any agent that can run shell commands can use Sidekick. MCP optional.

Your cloud sandbox has no Xcode or emulator. Use `sidekick` for all mobile work.

## Auth (one time)
- Get a key from console Settings → API keys (scoped to your default org, created at signup).
- `sidekick login` (paste it) or `export SIDEKICK_TOKEN=sk_live_...`. `sidekick org` shows your org.
- New machine: `sidekick agent-setup` installs this skill into ~/.claude/skills, then registers MCP: `claude mcp add sidekick -- sidekick mcp`. No MCP client? The CLI alone covers every verb below.
- Fastest live device: `sidekick dev` boots a sim and prints its share URL (add `--json` for scripts).

## Build iOS (no Mac needed)
- `sidekick xcode build . --scheme MyApp` — syncs code, streams xcodebuild logs. Iterate on errors.
- Warm cache is automatic (DerivedData + S3 asset storage). Upload `.app` once, reuse.

## Drive iOS simulator
- `sidekick ios create --device "iPhone 16, iOS 18.2"` → returns id (provisioning). Poll `sidekick ios get <id>` until `running`, then `webrtc_url` appears. Share webrtc_url as preview link on PR.
- `sidekick ios ax-tree <id>` first, then `sidekick ios act <id> --tap @e5` / `--type "hi"` / `--swipe up`. Batch file for ms precision: `--batch actions.json`. Poll the returned `action_ids` via the API until `done` (the daemon posts each result).
- `sidekick ios screenshot <id> --out shot.png`, `sidekick ios logs <id> --tail 100` for crashes.
- `sidekick ios record <id>` to start, `sidekick ios record <id> --stop --out demo.mp4` → mp4 demo video for PR.

## Drive Android
- No Mac needed, ever: emus run on Linux KVM hosts (`emu-host` boots AVDs per claim, self-host via `docker-compose.selfhost.yml`).
- `sidekick android create --profile Pixel_7` → id (provisioning). Poll `sidekick android get <id>` until `running`, then adb tunnel on localhost:5555: `adb-proxy --emu <id>` (uses SIDEKICK_TOKEN). Then plain `adb install app.apk`, `maestro test flow.yaml`, Appium, scrcpy all work.
- Same verbs: `android ui-tree` (@e refs from the on-device dump), `android screenshot --out shot.png`, `android logs --tail 100`, preview link.
- Media + posture over the tunnel: `android record --out demo.mp4 --duration 30s`, `android netcap --out cap.pcap --duration 30s`, `android rotate 90`, `android posture half-open` (foldables only).

## Rules
- Always build before creating a simulator. Always `ui-tree` before acting (refs go stale after each screen change).
- Screenshot when checking look (tree has no color/stacking). Logs when app crashes.
- Stop sessions when done: `sidekick ios delete <id>` / `sidekick android delete <id>` — streaming minutes bill until stopped.
- If asked only to "implement X", still verify on device and post preview link + video.
