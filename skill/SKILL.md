# OpenLim agent skill — drop into ~/.claude/skills/openlim/ or .claude/skills/
# Any agent that can run shell commands can use OpenLim. MCP optional.

Your cloud sandbox has no Xcode or emulator. Use `lim` for all mobile work.

## Build iOS (no Mac needed)
- `lim xcode build . --scheme MyApp` — syncs code, streams xcodebuild logs. Iterate on errors.
- Warm cache is automatic (DerivedData + S3 asset storage). Upload `.app` once, reuse.

## Drive iOS simulator
- `lim ios create --device "iPhone 16, iOS 18.2"` → returns id + webrtc_url + ax_tree_url. Share webrtc_url as preview link on PR.
- `lim ios ax-tree <id>` first, then `lim ios act <id> --tap @e5` / `--type "hi"` / `--swipe up`. Batch file for ms precision: `--batch actions.json`.
- `lim ios record <id> --stop` → mp4 demo video for PR. `lim ios logs <id> --tail 100` for crashes.

## Drive Android
- `lim android create --profile Pixel_7` → adb tunnel on localhost:5555. Then plain `adb install app.apk`, `maestro test flow.yaml`, Appium, scrcpy all work.
- Same verbs: ui-tree, screenshot, record, preview link.

## Rules
- Always build before creating a simulator. Always `ui-tree` before acting (refs go stale after each screen change).
- Screenshot when checking look (tree has no color/stacking). Logs when app crashes.
- Stop sessions when done: `lim ios delete <id>` / `lim android delete <id>` — streaming minutes bill until stopped.
- If asked only to "implement X", still verify on device and post preview link + video.
