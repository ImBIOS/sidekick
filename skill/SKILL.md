<!-- SPDX-License-Identifier: Apache-2.0 | Copyright 2026 ImBIOS — see LICENSE-APACHE.md -->
# Sidekick agent skill — drop into ~/.claude/skills/sidekick/ or .claude/skills/
# Any agent that can run shell commands can use Sidekick. MCP optional.

Your cloud sandbox has no Xcode or emulator. Use `sidekick` for all mobile work.

## Auth (one time)
- Get a key from console Settings → API keys (scoped to your default org, created at signup).
- `sidekick login` (paste it) or `export SIDEKICK_TOKEN=sk_live_...`. `sidekick org` shows your org.

## Build iOS (no Mac needed)
- `sidekick xcode build . --scheme MyApp` — syncs code, streams xcodebuild logs. Iterate on errors.
- Warm cache is automatic (DerivedData + S3 asset storage). Upload `.app` once, reuse.

## Drive iOS simulator
- `sidekick ios create --device "iPhone 16, iOS 18.2"` → returns id + webrtc_url + ax_tree_url. Share webrtc_url as preview link on PR.
- `sidekick ios ax-tree <id>` first, then `sidekick ios act <id> --tap @e5` / `--type "hi"` / `--swipe up`. Batch file for ms precision: `--batch actions.json`.
- `sidekick ios record <id> --stop` → mp4 demo video for PR. `sidekick ios logs <id> --tail 100` for crashes.

## Drive Android
- `sidekick android create --profile Pixel_7` → adb tunnel on localhost:5555. Then plain `adb install app.apk`, `maestro test flow.yaml`, Appium, scrcpy all work.
- Same verbs: ui-tree, screenshot, record, preview link.

## Rules
- Always build before creating a simulator. Always `ui-tree` before acting (refs go stale after each screen change).
- Screenshot when checking look (tree has no color/stacking). Logs when app crashes.
- Stop sessions when done: `sidekick ios delete <id>` / `sidekick android delete <id>` — streaming minutes bill until stopped.
- If asked only to "implement X", still verify on device and post preview link + video.
